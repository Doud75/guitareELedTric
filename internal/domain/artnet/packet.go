package artnet

import "encoding/binary"

func BuildArtNetHeader(universe int) []byte {
    header := make([]byte, 18)
    copy(header[0:8], []byte("Art-Net\x00"))
    binary.LittleEndian.PutUint16(header[8:10], 0x5000)
    binary.BigEndian.PutUint16(header[10:12], 14)
    header[12] = 0
    header[13] = 0
    binary.LittleEndian.PutUint16(header[14:16], uint16(universe))
    binary.BigEndian.PutUint16(header[16:18], 512)
    return header
}
