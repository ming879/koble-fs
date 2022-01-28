#!/bin/bash

CMDLINE_OPTS=""

# Remove IP addresses from interfaces
ifaces=( $(ip addr list | awk -F': ' '/^[0-9]/ && $2 != "lo" {print $2}') )
for eth in "${ifaces[@]}"; do
    iface=$(echo "${eth%@*}")
    eth_addr=$(ip -f inet addr show "$iface" | awk '/inet / {print $2}')
    eth_route=$(ip route show | grep "$iface" | head -n1 | awk '{print $3}')
    if [ "$eth_addr" != "" ]; then
        ip addr del "$eth_addr" dev "$iface"
        CMDLINE_OPTS+=" kstart:interfaces.${iface}=${eth_addr}"
        CMDLINE_OPTS+=" kstart:def_route=${iface}:${eth_route}"
    fi
done

# Set up mount points
mkdir /run/uml
rm -f /run/uml/machine.ready

# Start UML kernel
echo "running kernel with :: $@ $CMDLINE_OPTS"
exec "$@ $CMDLINE_OPTS"
