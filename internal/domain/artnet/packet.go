package artnet
import 	"encoding/binary" // On a besoin de cet outil pour écrire les en-têtes proprement


// buildArtNetHeader est une fonction privée qui construit un en-tête une seule fois.
// Elle est appelée par le constructeur.
func BuildArtNetHeader(universe int) []byte {
	header := make([]byte, 18)
	copy(header[0:8], []byte("Art-Net\x00"))          // Signature
	binary.LittleEndian.PutUint16(header[8:10], 0x5000) // OpCode ArtDmx
	binary.BigEndian.PutUint16(header[10:12], 14)       // Version du protocole
	header[12] = 0                                     // Sequence (non utilisé)
	header[13] = 0                                     // Physical Port (non utilisé)
	binary.LittleEndian.PutUint16(header[14:16], uint16(universe)) // Le numéro d'univers
	binary.BigEndian.PutUint16(header[16:18], 512) // La longueur des données
	return header
}