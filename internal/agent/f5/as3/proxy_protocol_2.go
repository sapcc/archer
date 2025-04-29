// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package as3

var pp2 = `
# serverside PROXY Protocol V2 implementation
# https://github.com/haproxy/haproxy/blob/ffdf6a32a7413d5bcf9223c3556b765b5e456a69/doc/proxy-protocol.txt

# converts an ipv4 ip in dotted notation into 4 binary strings with one byte in hex each
proc ip2hex { ip } {
    set octets [split [getfield $ip % 1] .]
    return [binary format c4 $octets]
}

# converts an 2Byte integer to 2 binary strings with one byte in hex each
proc port2hex { port } {
    return [binary format S $port]
}

proc proxy_addr {} {
    # https://github.com/haproxy/haproxy/blob/ffdf6a32a7413d5bcf9223c3556b765b5e456a69/doc/proxy-protocol.txt#L471-L488
    # proxy_addr
    clientside {
        return [call ip2hex [IP::remote_addr]][call ip2hex [IP::local_addr]][call port2hex [TCP::remote_port]][call port2hex [TCP::local_port]]
    }
}

proc tlv {binary_type binary_value} {
    # TLV: https://github.com/haproxy/haproxy/blob/ffdf6a32a7413d5bcf9223c3556b765b5e456a69/doc/proxy-protocol.txt#L525-L530
    # calculate length of data for length_hi and length_lo (2 byte field)
    set tlv_length_hilo [binary format S [string length $binary_value]]

    return $binary_type$tlv_length_hilo$binary_value
}

#proc tlv_type5 {} {
#    return [call tlv \x05 [binary format H* [lindex [AES::key 128] 2]]]
#}

proc tlv_sapcc {uuid4} {
    # PP2_TYPE_SAPCC 0xEC
    # prepare TLV
    # TLV: https://github.com/haproxy/haproxy/blob/ffdf6a32a7413d5bcf9223c3556b765b5e456a69/doc/proxy-protocol.txt#L525-L530
    return [call tlv \xec [binary format A* $uuid4]]
}

when SERVER_CONNECTED priority 900 {
    # create TLV type 5
    #set pp2_tlv [call tlv_type5]
    #set pp2_tlv [call tlv_sapcc {497f6eca-6276-4993-bfeb-53cbbbba6f08}]
    set pp2_tlv_strlen [string length [virtual name]]
    set pp2_tlv [call tlv_sapcc [string range [virtual name] [expr { $pp2_tlv_strlen - 36 }] $pp2_tlv_strlen]]

    # https://github.com/haproxy/haproxy/blob/ffdf6a32a7413d5bcf9223c3556b765b5e456a69/doc/proxy-protocol.txt#L335
    # proxy protocol version 2 signature (12 bytes)
    set pp2_header_signature \x0d\x0a\x0d\x0a\x00\x0d\x0a\x51\x55\x49\x54\x0a

    # https://github.com/haproxy/haproxy/blob/ffdf6a32a7413d5bcf9223c3556b765b5e456a69/doc/proxy-protocol.txt#L340-L358
    # proxy protocol version and command (\x2 -> proxy protocol version is 2; \x1 -> command is PROXY)
    #set pp2_header_byte13 [expr {(0x02 << 4) + 0x01}]
    set pp2_header_byte13 \x21

    # https://github.com/haproxy/haproxy/blob/ffdf6a32a7413d5bcf9223c3556b765b5e456a69/doc/proxy-protocol.txt#L360-L433
    # transport protocol and address family (\x1 -> transport protocol is STREAM/TCP; \x1 -> address family is AF_INET/ipv4)
    #set pp2_header_byte14 [expr {(0x01 << 4) + 0x01}]
    set pp2_header_byte14 \x11

    # proxy protocol addr
    set pp_proxy_addr [call proxy_addr]

    # proxy protocol length
    # number of following bytes part of the header in network endian order
    # https://github.com/haproxy/haproxy/blob/ffdf6a32a7413d5bcf9223c3556b765b5e456a69/doc/proxy-protocol.txt#L441-L449
    #                                           $pp_proxy_addr len is 12
    # NOTE: if the TLV length is static, it could be pre-calculated similar to pp_proxy_addr
    set pp2_header_byte1516 [binary format S [expr {12 + [string length $pp2_tlv]}]]

    # construct pp2_header
    set pp2_header ${pp2_header_signature}${pp2_header_byte13}${pp2_header_byte14}$pp2_header_byte1516$pp_proxy_addr$pp2_tlv

    # ensure pp header conversion of \x00 to NUL(0x00), not c080
    binary scan $pp2_header H* tmp

    # send pp
    TCP::respond $pp2_header
}`
