# TCP Segment Reordering

TCP segment reordering splits selected TCP payloads and sends altered packets so DPI devices must handle out-of-order and retransmitted data.

UA3F splits early TCP payloads into two parts, forwards the tail first, and drops the first part to create out-of-order delivery. The missing first part then triggers TCP retransmission, which can drive another round of reordered delivery.

`reorder-bytes` and `reorder-packets` control the scope and rounds of reordered transmission. Because this feature affects early packets after connection establishment, it can cause short-lived connection instability, but it should not affect long-running traffic after the early packets have passed.

```yaml
desync:
  reorder: true
  reorder-bytes: 8
  reorder-packets: 1500
```

UA3F skips packets without TCP payload, packets with payload length less than or equal to one byte, and packets with `FIN`.

This feature may affect the first packets of a connection. Test conservative values on the target network before broad deployment.

## Why reordering can affect DPI

DPI devices often reconstruct TCP streams and run protocol or signature matching on the reconstructed content. The goal is to inspect traffic as close as possible to the real endpoint conversation.

In practice, TCP stream reassembly is constrained by performance and implementation strategy. Abnormal TCP behavior such as out-of-order segments, retransmissions, sequence overlap, and window changes can significantly increase DPI complexity and cache pressure.

UA3F creates reordered fragments and retransmission-like behavior, which can produce these effects:

| Effect | Description |
| --- | --- |
| State divergence | DPI must track TCP sequence numbers and decide which bytes are authoritative when segments arrive out of order or overlap with retransmissions. Endpoint TCP stacks and simplified DPI logic may disagree, causing desynchronization. |
| Cache pressure | Fully buffering out-of-order streams is expensive at high throughput. Devices that timeout, drop buffered segments, or simplify reassembly may miss or misclassify content. |
| Protocol ambiguity | DPI often sniffs protocols from early packets. Transport-layer uncertainty can push parsers into the wrong detection path early in the connection. |
