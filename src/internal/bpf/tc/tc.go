//go:build linux

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux tc tc.c

package tc

import (
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

var skipInterfaces = map[string]bool{
	"lo":     true,
	"br-lan": true,
}

const (
	tcFilterPri = 1
	tcFilterHnd = 1
)

type classicAttachment struct {
	ifindex  int
	priority uint16
	handle   uint32
	name     string
}

type tcProgramAttachment struct {
	name    string
	program *ebpf.Program
}

type TC struct {
	objs         *tcObjects
	links        []link.Link // TCX links (kernel >= 6.6)
	classicLinks []classicAttachment
}

func NewTC(cfg *config.L3RewriteConfig) (*TC, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("remove memlock: %w", err)
	}

	var objs tcObjects
	if err := loadTcObjects(&objs, nil); err != nil {
		return nil, fmt.Errorf("load tc objs: %w", err)
	}

	programs := selectedPrograms(&objs, cfg)
	if len(programs) == 0 {
		objs.Close()
		return nil, fmt.Errorf("no TC program selected")
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		objs.Close()
		return nil, fmt.Errorf("list interfaces: %w", err)
	}

	defaultRouteIfaces, err := getIPv4DefaultRouteInterfaces()
	if err != nil {
		objs.Close()
		return nil, fmt.Errorf("list ipv4 default routes: %w", err)
	}

	var eligible []net.Interface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if skipInterfaces[iface.Name] {
			continue
		}
		if !isEthernetInterface(iface) {
			slog.Info("Skip interface for TC", "name", iface.Name, "index", iface.Index, "reason", "non-ethernet")
			continue
		}
		if !hasIPv4Address(iface) {
			slog.Info("Skip interface for TC", "name", iface.Name, "index", iface.Index, "reason", "no ipv4 address")
			continue
		}
		if !defaultRouteIfaces[iface.Index] {
			slog.Info("Skip interface for TC", "name", iface.Name, "index", iface.Index, "reason", "not on ipv4 default route")
			continue
		}
		eligible = append(eligible, iface)
		slog.Info("Eligible interface for TC", "name", iface.Name, "index", iface.Index)
	}

	if len(eligible) == 0 {
		objs.Close()
		return nil, fmt.Errorf("no eligible interfaces for TC")
	}

	// Try TCX first (kernel >= 6.6), fall back to cls_bpf on older kernels.
	links, err := attachAllTCX(eligible, programs)
	var classicLinks []classicAttachment
	if err != nil {
		slog.Warn("TCX attach failed, falling back to classic cls_bpf", "error", err)
		classicLinks, err = attachAllClassic(eligible, programs)
	}
	if err != nil {
		objs.Close()
		return nil, err
	}

	return &TC{objs: &objs, links: links, classicLinks: classicLinks}, nil
}

func isEthernetInterface(iface net.Interface) bool {
	lnk, err := netlink.LinkByIndex(iface.Index)
	if err != nil {
		return false
	}
	attrs := lnk.Attrs()
	if attrs == nil {
		return false
	}
	return attrs.EncapType == "ether"
}

func hasIPv4Address(iface net.Interface) bool {
	addrs, err := iface.Addrs()
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			if v.IP != nil && v.IP.To4() != nil {
				return true
			}
		case *net.IPAddr:
			if v.IP != nil && v.IP.To4() != nil {
				return true
			}
		}
	}
	return false
}

func getIPv4DefaultRouteInterfaces() (map[int]bool, error) {
	routes, err := netlink.RouteList(nil, unix.AF_INET)
	if err != nil {
		return nil, err
	}

	defaultIfaces := make(map[int]bool)
	for _, route := range routes {
		if !isIPv4DefaultRoute(route.Dst) {
			continue
		}
		defaultIfaces[route.LinkIndex] = true
	}

	return defaultIfaces, nil
}

func isIPv4DefaultRoute(dst *net.IPNet) bool {
	// Linux may represent default IPv4 route as nil or as 0.0.0.0/0.
	if dst == nil {
		return true
	}
	ones, bits := dst.Mask.Size()
	if bits != 32 || ones != 0 {
		return false
	}
	ip4 := dst.IP.To4()
	if ip4 == nil {
		return false
	}
	return ip4.Equal(net.IPv4zero)
}

func (t *TC) Close() error {
	if t == nil {
		return nil
	}

	var errs []error

	for _, l := range t.links {
		if err := l.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	t.links = nil

	for _, attachment := range t.classicLinks {
		if err := deleteClsBpfFilter(attachment); err != nil {
			errs = append(errs, fmt.Errorf("delete cls_bpf filter %s on ifindex %d: %w", attachment.name, attachment.ifindex, err))
		}
	}
	t.classicLinks = nil

	if t.objs != nil {
		if err := t.objs.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close tc objs: %w", err))
		}
		t.objs = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("tc cleanup: %w", errors.Join(errs...))
	}
	return nil
}

func selectedPrograms(objs *tcObjects, cfg *config.L3RewriteConfig) []tcProgramAttachment {
	var programs []tcProgramAttachment

	if cfg.IPID {
		programs = append(programs, tcProgramAttachment{name: "set_ip_id_zero", program: objs.SetIpIdZero})
		slog.Info("Selected TC program", "Set IP ID", true)
	}
	if cfg.TTL {
		programs = append(programs, tcProgramAttachment{name: "set_ip_ttl", program: objs.SetIpTtl})
		slog.Info("Selected TC program", "Set TTL", true)
	}
	if cfg.TCPWIN {
		programs = append(programs, tcProgramAttachment{name: "set_tcp_syn_window", program: objs.SetTcpSynWindow})
		slog.Info("Selected TC program", "Set TCP Initial Window", true)
	}
	if cfg.TCPTS {
		programs = append(programs, tcProgramAttachment{name: "clear_tcp_syn_ts", program: objs.ClearTcpSynTs})
		slog.Info("Selected TC program", "Clear TCP Timestamp", true)
	}

	return programs
}

// attachAllTCX attaches the BPF program to the egress hook of each eligible
// interface using the TCX path (kernel >= 6.6). On any failure the successfully
// attached links are closed before returning.
func attachAllTCX(ifaces []net.Interface, programs []tcProgramAttachment) ([]link.Link, error) {
	var links []link.Link
	for _, iface := range ifaces {
		for _, program := range programs {
			l, err := link.AttachTCX(link.TCXOptions{
				Interface: iface.Index,
				Program:   program.program,
				Attach:    ebpf.AttachTCXEgress,
			})
			if err != nil {
				for _, existing := range links {
					existing.Close()
				}
				return nil, fmt.Errorf("attach tcx egress program %s on %s: %w", program.name, iface.Name, err)
			}
			links = append(links, l)
		}
	}
	return links, nil
}

// attachAllClassic attaches the BPF program (identified by progFD) to the
// egress hook of each eligible interface using the classic cls_bpf path.
// On any failure the successfully attached filters are removed before returning.
func attachAllClassic(ifaces []net.Interface, programs []tcProgramAttachment) ([]classicAttachment, error) {
	var attached []classicAttachment
	for _, iface := range ifaces {
		if err := ensureClsactQdisc(iface.Index); err != nil {
			for _, existing := range attached {
				_ = deleteClsBpfFilter(existing)
			}
			return nil, fmt.Errorf("ensure clsact qdisc on %s: %w", iface.Name, err)
		}
		for index, program := range programs {
			attachment := classicAttachment{
				ifindex:  iface.Index,
				priority: uint16(tcFilterPri + index),
				handle:   uint32(tcFilterHnd + index),
				name:     fmt.Sprintf("%s:%s", iface.Name, program.name),
			}
			if err := addClsBpfFilter(attachment, program.program.FD()); err != nil {
				for _, existing := range attached {
					_ = deleteClsBpfFilter(existing)
				}
				return nil, fmt.Errorf("add cls_bpf filter %s on %s: %w", program.name, iface.Name, err)
			}
			attached = append(attached, attachment)
		}
	}
	return attached, nil
}

// ensureClsactQdisc adds a clsact qdisc to the interface if one does not exist.
func ensureClsactQdisc(ifindex int) error {
	qdisc := &netlink.Clsact{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: ifindex,
			Handle:    netlink.MakeHandle(0xffff, 0),
			Parent:    netlink.HANDLE_CLSACT,
		},
	}
	err := netlink.QdiscAdd(qdisc)
	if errors.Is(err, unix.EEXIST) {
		return nil
	}
	return err
}

// addClsBpfFilter attaches progFD as a direct-action cls_bpf filter on the
// egress hook of ifindex.
func addClsBpfFilter(attachment classicAttachment, progFD int) error {
	filter := &netlink.BpfFilter{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: attachment.ifindex,
			Handle:    attachment.handle,
			Parent:    netlink.HANDLE_MIN_EGRESS,
			Priority:  attachment.priority,
			Protocol:  unix.ETH_P_IP,
		},
		Fd:           progFD,
		Name:         attachment.name,
		DirectAction: true,
	}
	return netlink.FilterAdd(filter)
}

// deleteClsBpfFilter removes the cls_bpf filter previously added by
// addClsBpfFilter on ifindex. ENOENT is treated as success.
func deleteClsBpfFilter(attachment classicAttachment) error {
	filter := &netlink.BpfFilter{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: attachment.ifindex,
			Handle:    attachment.handle,
			Parent:    netlink.HANDLE_MIN_EGRESS,
			Priority:  attachment.priority,
			Protocol:  unix.ETH_P_IP,
		},
	}
	err := netlink.FilterDel(filter)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	return err
}
