// Package web menyimpan hasil build antarmuka React dan menanamkannya ke dalam binary.
//
// Folder dist diisi saat proses build container: tahap Node menjalankan Vite lebih dulu,
// hasilnya disalin ke sini, barulah binary Go dikompilasi. Dengan cara ini satu container
// sudah memuat antarmuka dan backend sekaligus tanpa perlu server berkas terpisah.
package web

import "embed"

//go:embed all:dist
var DistFS embed.FS
