include $(TOPDIR)/rules.mk

PKG_NAME:=UA3F
PKG_VERSION:=1.6.0
PKG_RELEASE:=1

# PKG_SOURCE:=$(PKG_NAME)-$(PKG_VERSION).tar.gz
# PKG_SOURCE_URL:=https://codeload.github.com/SunBK201/UA3F/tar.gz/v$(PKG_VERSION)?
# PKG_HASH:=660a89fbde16ec769f256e83dc9479bf3fb2309e0c26fe88b3e4e55fbf6fbf8d

PKG_MAINTAINER:=SunBK201 <sunbk201gm@gmail.com>
PKG_LICENSE:=GPL-3.0-only
PKG_LICENSE_FILES:=LICENSE

PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_PARALLEL:=1
PKG_BUILD_FLAGS:=no-mips16

GO_PKG:=github.com/sunbk201/ua3f
GO_PKG_LDFLAGS_X:= main.version=$(PKG_VERSION)

include $(INCLUDE_DIR)/package.mk
include $(TOPDIR)/feeds/packages/lang/golang/golang-package.mk

define Package/ua3f
	SECTION:=net
	CATEGORY:=Network
	SUBMENU:=Web Servers/Proxies
	TITLE:=A SOCKS5 Server for User-Agent Rewriting
	URL:=https://github.com/SunBK201/UA3F
	DEPENDS:=$(GO_ARCH_DEPENDS) +luci-compat +ipset +iptables +iptables-mod-tproxy +iptables-mod-extra +iptables-mod-nat-extra +kmod-ipt-conntrack +iptables-mod-ipopt +iptables-mod-nfqueue +iptables-mod-conntrack-extra +kmod-nf-conntrack-netlink
endef

define Package/ua3f/description
	Advanced HTTP User-Agent Rewriting Tool.
endef

define Build/Prepare
	$(CP) ../src/* $(PKG_BUILD_DIR)
	po2lmo ./po/zh_cn/ua3f.po $(PKG_BUILD_DIR)/ua3f.zh-cn.lmo
endef

define Package/ua3f/conffiles
/etc/config/ua3f
endef

define Package/ua3f/install
	$(call GoPackage/Package/Install/Bin,$(PKG_INSTALL_DIR))

	$(INSTALL_DIR) $(1)/usr/bin/
	# $(INSTALL_BIN) $(GO_PKG_BUILD_BIN_DIR)/ua3f $(1)/usr/bin/
	$(INSTALL_BIN) $(PKG_INSTALL_DIR)/usr/bin/ua3f $(1)/usr/bin/
	$(INSTALL_DIR) $(1)/etc/init.d/
	$(INSTALL_BIN) ./files/ua3f.init $(1)/etc/init.d/ua3f
	$(INSTALL_DIR) $(1)/etc/config/
	$(INSTALL_CONF) ./files/ua3f.uci $(1)/etc/config/ua3f
	$(INSTALL_DIR) $(1)/usr/lib/lua/luci/i18n/
	$(INSTALL_DATA) $(PKG_BUILD_DIR)/ua3f.zh-cn.lmo $(1)/usr/lib/lua/luci/i18n/ua3f.zh-cn.lmo
	$(CP) $(PKG_BUILD_DIR)/files/luci/* $(1)/usr/lib/lua/luci/
endef

$(eval $(call GoBinPackage,ua3f))
$(eval $(call BuildPackage,ua3f))
