#!/bin/bash
set -eu

echo "Configs:"

echo "host: $host"
echo "port: $port"
echo "proto: $proto"
echo "ca_crt: $(if [ ! -z $ca_crt ]; then echo "***"; fi)"
echo "client_crt: $(if [ ! -z $client_crt ]; then echo "***"; fi)"
echo "client_key: $(if [ ! -z $client_key ]; then echo "***"; fi)"

echo ""

log_path=$(mktemp)

case "$OSTYPE" in
  linux*)
    echo "Configuring for Ubuntu"

    echo ${ca_crt} | base64 -d > /etc/openvpn/ca.crt
    echo ${client_crt} | base64 -d > /etc/openvpn/client.crt
    echo ${client_key} | base64 -d > /etc/openvpn/client.key

    cat <<EOF > /etc/openvpn/client.conf
client
dev tun
proto ${proto}
remote ${host} ${port}
resolv-retry infinite
nobind
persist-key
persist-tun
comp-lzo
verb 3
ca ca.crt
cert client.crt
key client.key
EOF

    service openvpn start client > $log_path 2>&1
    sleep 5

    if ifconfig | grep tun0 > /dev/null
    then
      echo "VPN tunnel opened successfully"
      echo ""
      echo "Test VPN connection"
      if ping -c 1 -t 5 10.1.0.1 ; then
        echo "Connection alive"
      else
        echo "Failed to connect, error:"
        cat $log_path
        exit 1
      fi
    else
      echo "VPN opening tunnel failed!"
      exit 1
    fi
    ;;
  darwin*)
    echo "Configuring for Mac OS"

    echo ${ca_crt} | base64 -D -o ca.crt > /dev/null 2>&1
    echo ${client_crt} | base64 -D -o client.crt > /dev/null 2>&1
    echo ${client_key} | base64 -D -o client.key > /dev/null 2>&1

    sudo openvpn --client --dev tun --proto ${proto} --remote ${host} ${port} --resolv-retry infinite --nobind --persist-key --persist-tun --comp-lzo --verb 3 --ca ca.crt --cert client.crt --key client.key > $log_path 2>&1 &

    sleep 5

    if ifconfig -l | grep utun0 > /dev/null
    then
      echo "VPN tunnel opened successfully"
      echo ""
      echo "Test VPN connection"
      if ping -c 1 -t 5 10.1.0.1 ; then
        echo "Connection alive"
      else
        echo "Failed to connect, error:"
        cat $log_path
        exit 1
      fi
    else
      echo "VPN opening tunnel failed!"
      exit 1
    fi
    ;;
  *)
    echo "Unknown operative system: $OSTYPE, exiting"
    exit 1
    ;;
esac
