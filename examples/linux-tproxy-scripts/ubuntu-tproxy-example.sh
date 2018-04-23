#!/usr/bin/bash

echo "Loading modules.."
modprobe -a nf_tproxy_core xt_TPROXY xt_socket xt_mark ip_gre gre


LOCALIP="10.80.2.2"
CISCODIRIP="10.80.2.1"
CISCOIPID="192.168.10.127"

echo "changing routing and reverse path stuff.."
echo 0 > /proc/sys/net/ipv4/conf/lo/rp_filter
echo 1 > /proc/sys/net/ipv4/ip_forward

echo "creating tunnel..."
iptunnel add wccp0 mode gre remote $CISCOIPID local $LOCALIP dev eth1
ifconfig wccp0 127.0.1.1/32 up

echo "creating routing table for tproxy..."
ip rule add fwmark 1 lookup 100
ip route add local 0.0.0.0/0 dev lo table 100

echo "creating iptables tproxy rules..."
iptables -A INPUT  -i lo -j ACCEPT
iptables -A INPUT  -p icmp -m icmp --icmp-type any -j ACCEPT
iptables -A FORWARD -i lo -j ACCEPT
iptables -A INPUT  -s $CISCODIRIP -p udp -m udp --dport 2048 -j ACCEPT
iptables -A INPUT -i wccp0 -j ACCEPT
iptables -A INPUT -p gre -j ACCEPT

iptables -t mangle -F
iptables -t mangle -A PREROUTING -d $LOCALIP -j ACCEPT
iptables -t mangle -N DIVERT
iptables -t mangle -A DIVERT -j MARK --set-mark 1
iptables -t mangle -A DIVERT -j ACCEPT
iptables -t mangle -A PREROUTING -p tcp -m socket -j DIVERT
iptables -t mangle -A PREROUTING -p tcp --dport 80 -j TPROXY --tproxy-mark 0x1/0x1 --on-port 3129
