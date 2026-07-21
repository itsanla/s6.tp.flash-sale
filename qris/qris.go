// Package qris menyusun payload QRIS dan merender gambar QR untuk pembayaran.
//
// PENTING: seluruh isi paket ini adalah SIMULASI untuk keperluan demonstrasi akademik.
// Identitas merchant yang dipakai bukan merchant terdaftar, sehingga QR yang dihasilkan
// tidak dapat dan tidak boleh dipakai untuk transaksi pembayaran sungguhan. Pelunasan
// pembayaran pada aplikasi ini dilakukan lewat halaman uji /test/qris-list.
package qris

import (
	"encoding/base64"
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// Generator menyusun payload QRIS memakai identitas merchant simulasi.
type Generator struct {
	MerchantName string
	MerchantCity string
	MerchantID   string
}

func NewGenerator(name, city, id string) *Generator {
	return &Generator{MerchantName: name, MerchantCity: city, MerchantID: id}
}

// tlv membentuk satu elemen berformat tag, panjang, lalu nilai sesuai standar EMVCo.
func tlv(tag, value string) string {
	return fmt.Sprintf("%s%02d%s", tag, len(value), value)
}

// crc16 menghitung checksum CRC-16/CCITT-FALSE yang dipakai sebagai penutup payload QRIS.
func crc16(data string) uint16 {
	crc := uint16(0xFFFF)
	for i := 0; i < len(data); i++ {
		crc ^= uint16(data[i]) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func sanitize(s string, max int) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) > max {
		s = s[:max]
	}
	return s
}

// BuildPayload menyusun string QRIS dinamis untuk satu order beserta nominalnya.
func (g *Generator) BuildPayload(orderCode string, amount int64) string {
	merchantAccount := tlv("00", "ID.CO.QRIS.WWW") +
		tlv("01", g.MerchantID) +
		tlv("02", sanitize(orderCode, 25)) +
		tlv("03", "UMI")

	additional := tlv("01", sanitize(orderCode, 25))

	var sb strings.Builder
	sb.WriteString(tlv("00", "01"))                            // versi format payload
	sb.WriteString(tlv("01", "12"))                            // 12 berarti QR dinamis sekali pakai
	sb.WriteString(tlv("26", merchantAccount))                 // informasi merchant domestik
	sb.WriteString(tlv("52", "7996"))                          // kode kategori merchant: taman hiburan
	sb.WriteString(tlv("53", "360"))                           // mata uang rupiah
	sb.WriteString(tlv("54", fmt.Sprintf("%d", amount)))       // nominal transaksi
	sb.WriteString(tlv("58", "ID"))                            // kode negara
	sb.WriteString(tlv("59", sanitize(g.MerchantName, 25)))    // nama merchant
	sb.WriteString(tlv("60", sanitize(g.MerchantCity, 15)))    // kota merchant
	sb.WriteString(tlv("62", additional))                      // data tambahan berisi kode order

	body := sb.String() + "6304"
	return body + fmt.Sprintf("%04X", crc16(body))
}

// RenderPNGBase64 mengubah payload menjadi gambar QR berformat PNG dalam bentuk data URI,
// sehingga frontend cukup menampilkannya lewat atribut src tanpa permintaan tambahan.
func RenderPNGBase64(payload string, size int) (string, error) {
	png, err := qrcode.Encode(payload, qrcode.Medium, size)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}
