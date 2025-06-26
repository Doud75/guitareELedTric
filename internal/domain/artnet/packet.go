package artnet

func Build(universe int, data [512]byte) []byte {
    packet := make([]byte, 18+512)
    copy(packet[0:], []byte("Art-Net\x00"))
    packet[8] = 0x00
    packet[9] = 0x50
    packet[10] = 0x00
    packet[11] = 14
    packet[12] = 0x00
    packet[13] = 0x00
    packet[14] = byte(universe & 0xFF)
    packet[15] = byte((universe >> 8) & 0xFF)
    packet[16] = byte(512 >> 8)
    packet[17] = byte(512 & 0xFF)
    copy(packet[18:], data[:])
    return packet
}
