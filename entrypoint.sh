#!/bin/bash

# Set up networking for UML taps
ifaces=( $(ip addr list | awk -F': ' '/^[0-9]/ && $2 != "lo" {print $2}') )
for eth in "${ifaces[@]}"; do
    iface=$(echo "${eth%@*}")
    echo "setting up network for $iface"
    num=$(echo "$iface" | tr -dc '0-9')
    tap="nk${num}"
    bridge="br${num}"
    ip tuntap add dev "${tap}" mode tap
    brctl addbr "${bridge}"
    brctl addif "${bridge}" "${iface}"
    brctl addif "${bridge}" "${tap}"

    ip link set "$iface" up
    ip link set "${tap}" up
    ip link set "${bridge}" up

    brctl stp "${bridge}" on
    brctl setageing "${bridge}" 1
    brctl setfd "${bridge}" 0

    eth_addr=$(ip -f inet addr show "$iface" | awk '/inet / {print $2}')
    eth_route=$(ip route show | grep "$iface" | head -n1 | awk '{print $3}')
    ip addr del "$eth_addr" dev "$iface"
    ip addr add "$eth_addr" dev "$bridge"
    ip route add default via "$eth_route" dev "$bridge"
    iptables -t nat -A POSTROUTING -o "$bridge" -j MASQUERADE
done

# Set up mount points
mkdir /run/uml
rm -f /run/uml/machine.ready

# Start UML kernel
echo "running kernel with :: $@"
exec "$@"
