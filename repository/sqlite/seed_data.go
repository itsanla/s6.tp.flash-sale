package sqlite

import "wahanapark/domain"

// img menyusun alamat foto wahana dari layanan gambar publik Unsplash.
//
// Foto dipakai sebagai gambar sementara untuk keperluan tampilan. Parameter ukuran dan
// mutu sengaja disetel agar berkas tetap ringan saat dimuat pada daftar katalog.
func img(id string) string {
	return "https://images.unsplash.com/photo-" + id + "?auto=format&fit=crop&w=900&q=70"
}

// seedData adalah katalog awal Taman Wahana Nusantara: 32 wahana yang tersebar pada
// enam kategori, mulai dari wahana ekstrem sampai wahana anak dan indoor.
var seedData = []domain.Ride{
	// ---------------------------------------------------------------- Ekstrem
	{
		Slug: "halilintar-petir", Name: "Halilintar Petir", Category: domain.CategoryEkstrem,
		Tagline:     "Roller coaster loop ganda dengan kecepatan 85 km/jam",
		Description: "Wahana andalan taman dengan lintasan sepanjang 900 meter, dua loop vertikal, dan satu corkscrew. Kereta melaju menembus terowongan berkabut sebelum menukik dari ketinggian 32 meter. Disarankan hanya untuk pengunjung yang benar benar menyukai tantangan.",
		Emoji:       "🎢", ImageURL: img("1627035983655-0ceec61bb733"), Price: 75000, DurationMin: 4, MinHeightCm: 140, ThrillLevel: 5, DailyQuota: 400,
	},
	{
		Slug: "ular-besi-terbalik", Name: "Ular Besi Terbalik", Category: domain.CategoryEkstrem,
		Tagline:     "Coaster gantung dengan kaki menggantung bebas",
		Description: "Duduk pada kursi gantung tanpa lantai, kaki melayang bebas sepanjang lintasan berkelok. Terdapat tiga inversi dan satu segmen zero gravity roll yang membuat tubuh terasa melayang selama dua detik penuh.",
		Emoji:       "⛓️", ImageURL: img("1621445944472-f252571005b6"), Price: 80000, DurationMin: 4, MinHeightCm: 140, ThrillLevel: 5, DailyQuota: 350,
	},
	{
		Slug: "menara-hysteria", Name: "Menara Hysteria", Category: domain.CategoryEkstrem,
		Tagline:     "Jatuh bebas dari ketinggian 45 meter",
		Description: "Kursi dinaikkan perlahan sampai puncak menara sambil memperlihatkan panorama seluruh taman, lalu dijatuhkan tanpa aba aba. Sensasi tanpa bobot terasa sekitar dua detik sebelum sistem rem magnetik bekerja dengan halus.",
		Emoji:       "🗼", ImageURL: img("1532262920791-bd095e2d7a9e"), Price: 65000, DurationMin: 2, MinHeightCm: 130, ThrillLevel: 5, DailyQuota: 300,
	},
	{
		Slug: "tornado-spin", Name: "Tornado Spin", Category: domain.CategoryEkstrem,
		Tagline:     "Berputar 360 derajat pada dua sumbu sekaligus",
		Description: "Lengan raksasa mengayun sambil memutar gondola pada porosnya sendiri, menghasilkan kombinasi putaran yang sulit ditebak. Setiap perjalanan memberi pola putaran yang berbeda beda.",
		Emoji:       "🌪️", ImageURL: img("1502137840985-4ec07e8568bf"), Price: 55000, DurationMin: 3, MinHeightCm: 130, ThrillLevel: 4, DailyQuota: 350,
	},
	{
		Slug: "kora-kora-samudra", Name: "Kora Kora Samudra", Category: domain.CategoryEkstrem,
		Tagline:     "Perahu bajak laut yang mengayun sampai 70 derajat",
		Description: "Perahu raksasa berkapasitas 48 penumpang mengayun semakin tinggi hingga hampir tegak lurus. Kursi paling belakang memberikan sensasi melayang paling terasa saat perahu mencapai titik tertinggi.",
		Emoji:       "🏴‍☠️", ImageURL: img("1502137914655-3ab2fb4dc4cc"), Price: 45000, DurationMin: 5, MinHeightCm: 120, ThrillLevel: 4, DailyQuota: 500,
	},
	{
		Slug: "ayunan-langit", Name: "Ayunan Langit", Category: domain.CategoryEkstrem,
		Tagline:     "Ayunan berputar pada ketinggian 30 meter",
		Description: "Kursi ayunan diangkat sambil berputar hingga ketinggian 30 meter, memberi pemandangan taman dari atas sambil merasakan angin bebas. Kombinasi antara menegangkan dan menyenangkan.",
		Emoji:       "🎠", ImageURL: img("1502136969935-8d8eef54d77b"), Price: 50000, DurationMin: 4, MinHeightCm: 120, ThrillLevel: 4, DailyQuota: 400,
	},

	// --------------------------------------------------------------- Keluarga
	{
		Slug: "bianglala-cakrawala", Name: "Bianglala Cakrawala", Category: domain.CategoryKeluarga,
		Tagline:     "Kincir raksasa 55 meter dengan gondola berpendingin",
		Description: "Ikon taman yang terlihat dari kejauhan. Setiap gondola tertutup dan berpendingin udara, memuat hingga enam orang. Pada sore hari pengunjung dapat menyaksikan matahari terbenam dari titik tertinggi.",
		Emoji:       "🎡", ImageURL: img("1598947720689-d7f934bde9e3"), Price: 40000, DurationMin: 12, MinHeightCm: 0, ThrillLevel: 1, DailyQuota: 600,
	},
	{
		Slug: "komidi-putar-kencana", Name: "Komidi Putar Kencana", Category: domain.CategoryKeluarga,
		Tagline:     "Kuda kayu klasik berlapis ornamen emas",
		Description: "Komidi putar dua tingkat dengan 48 kuda kayu berukir tangan dan empat kereta kencana untuk pengunjung yang membawa balita. Diiringi musik orkestra klasik sepanjang perjalanan.",
		Emoji:       "🎠", ImageURL: img("1597172984973-fa1a221fe91d"), Price: 30000, DurationMin: 5, MinHeightCm: 0, ThrillLevel: 1, DailyQuota: 700,
	},
	{
		Slug: "mobil-tabrakan", Name: "Mobil Tabrakan", Category: domain.CategoryKeluarga,
		Tagline:     "Arena bumper car beralas lantai licin",
		Description: "Arena seluas 600 meter persegi dengan 30 mobil listrik berbumper karet. Cocok untuk keluarga yang ingin saling berkejaran. Anak di bawah 100 sentimeter dapat ikut bila didampingi orang dewasa.",
		Emoji:       "🚗", ImageURL: img("1544441452-326ff5a947fd"), Price: 35000, DurationMin: 6, MinHeightCm: 100, ThrillLevel: 2, DailyQuota: 500,
	},
	{
		Slug: "kereta-wisata-keliling", Name: "Kereta Wisata Keliling", Category: domain.CategoryKeluarga,
		Tagline:     "Berkeliling taman dengan kereta beratap terbuka",
		Description: "Kereta wisata mengelilingi seluruh area taman sepanjang 2,4 kilometer, berhenti di lima stasiun utama. Pemandu memberikan penjelasan singkat mengenai setiap zona wahana yang dilewati.",
		Emoji:       "🚂", ImageURL: img("1589197471564-8266ed7f59b5"), Price: 25000, DurationMin: 15, MinHeightCm: 0, ThrillLevel: 1, DailyQuota: 800,
	},
	{
		Slug: "perahu-angsa-danau", Name: "Perahu Angsa Danau", Category: domain.CategoryKeluarga,
		Tagline:     "Mengayuh perahu angsa di danau buatan",
		Description: "Danau buatan seluas dua hektar dengan perahu angsa berkapasitas empat orang yang dikayuh sendiri. Tersedia jalur mengelilingi pulau kecil di tengah danau yang dihuni angsa dan bebek.",
		Emoji:       "🦢", ImageURL: img("1683806906116-99d37234b21c"), Price: 35000, DurationMin: 20, MinHeightCm: 0, ThrillLevel: 1, DailyQuota: 300,
	},
	{
		Slug: "rumah-cermin-labirin", Name: "Rumah Cermin Labirin", Category: domain.CategoryKeluarga,
		Tagline:     "Labirin cermin dengan 120 panel pemantul",
		Description: "Ruangan penuh cermin yang menciptakan ilusi lorong tak berujung. Terdapat empat zona dengan tingkat kesulitan berbeda, termasuk zona cermin cembung yang mengubah bentuk bayangan tubuh.",
		Emoji:       "🪞", ImageURL: img("1498736297812-3a08021f206f"), Price: 30000, DurationMin: 15, MinHeightCm: 0, ThrillLevel: 2, DailyQuota: 400,
	},

	// ------------------------------------------------------------------- Anak
	{
		Slug: "istana-balon-ceria", Name: "Istana Balon Ceria", Category: domain.CategoryAnak,
		Tagline:     "Istana tiup raksasa dengan perosotan empuk",
		Description: "Area bermain tiup seluas 400 meter persegi berbentuk istana, lengkap dengan perosotan, terowongan, dan dinding panjat lunak. Diawasi petugas dan dibatasi maksimal 40 anak per sesi.",
		Emoji:       "🏰", ImageURL: img("1582569789410-49a05e8a461f"), Price: 25000, DurationMin: 20, MinHeightCm: 0, ThrillLevel: 1, DailyQuota: 500,
	},
	{
		Slug: "kolam-bola-pelangi", Name: "Kolam Bola Pelangi", Category: domain.CategoryAnak,
		Tagline:     "Kolam berisi 50 ribu bola warna warni",
		Description: "Kolam bola dengan kedalaman aman 60 sentimeter, dilengkapi jaring pengaman dan area khusus balita. Bola dibersihkan setiap hari menggunakan mesin sterilisasi otomatis.",
		Emoji:       "🔴", ImageURL: img("1589374248297-8275f45b30e1"), Price: 25000, DurationMin: 30, MinHeightCm: 0, ThrillLevel: 1, DailyQuota: 500,
	},
	{
		Slug: "kuda-poni-mini", Name: "Kuda Poni Mini", Category: domain.CategoryAnak,
		Tagline:     "Menunggang poni jinak dengan pendamping",
		Description: "Anak dapat menunggang kuda poni terlatih mengelilingi arena berpasir sepanjang 80 meter. Setiap anak didampingi satu pawang, dan tersedia helm pengaman berbagai ukuran.",
		Emoji:       "🐴", ImageURL: img("1609828352150-34988b1685ea"), Price: 20000, DurationMin: 5, MinHeightCm: 0, ThrillLevel: 1, DailyQuota: 400,
	},
	{
		Slug: "kereta-mini-anak", Name: "Kereta Mini Anak", Category: domain.CategoryAnak,
		Tagline:     "Kereta kecil mengelilingi taman bunga",
		Description: "Kereta berukuran mini dengan lokomotif bergaya klasik, melintasi taman bunga dan terowongan mini. Orang tua dapat ikut duduk mendampingi anak di gerbong yang sama.",
		Emoji:       "🚞", ImageURL: img("1579963405196-8f694d063749"), Price: 20000, DurationMin: 8, MinHeightCm: 0, ThrillLevel: 1, DailyQuota: 450,
	},

	// ------------------------------------------------------------ Wahana Air
	{
		Slug: "seluncur-air-raksasa", Name: "Seluncur Air Raksasa", Category: domain.CategoryAir,
		Tagline:     "Empat jalur seluncur setinggi 18 meter",
		Description: "Terdiri dari jalur lurus berkecepatan tinggi, jalur spiral tertutup, jalur berbentuk mangkuk raksasa, dan jalur balap empat lintasan. Setiap jalur memiliki karakter luncuran yang berbeda.",
		Emoji:       "🛝", ImageURL: img("1701361650313-9b20b1d76820"), Price: 60000, DurationMin: 2, MinHeightCm: 120, ThrillLevel: 4, DailyQuota: 500,
	},
	{
		Slug: "kolam-ombak-samudra", Name: "Kolam Ombak Samudra", Category: domain.CategoryAir,
		Tagline:     "Ombak buatan setinggi 1,5 meter setiap 10 menit",
		Description: "Kolam seluas 1.800 meter persegi dengan mesin pembangkit ombak yang menghasilkan pola gelombang berbeda setiap sesi. Bagian tepi kolam sangat dangkal sehingga tetap aman untuk anak.",
		Emoji:       "🌊", ImageURL: img("1739295194212-0602c4d1e797"), Price: 55000, DurationMin: 60, MinHeightCm: 0, ThrillLevel: 3, DailyQuota: 800,
	},
	{
		Slug: "arung-jeram-log-flume", Name: "Arung Jeram Log Flume", Category: domain.CategoryAir,
		Tagline:     "Perahu kayu menukik dari ketinggian 14 meter",
		Description: "Perahu berbentuk batang kayu menyusuri kanal air melewati gua bertema hutan tropis, lalu menukik tajam di akhir lintasan dan menciptakan cipratan air setinggi lima meter.",
		Emoji:       "🛶", ImageURL: img("1703167906241-03f513c604fb"), Price: 65000, DurationMin: 6, MinHeightCm: 110, ThrillLevel: 4, DailyQuota: 450,
	},
	{
		Slug: "sungai-santai", Name: "Sungai Santai", Category: domain.CategoryAir,
		Tagline:     "Mengapung santai sepanjang 400 meter",
		Description: "Sungai buatan dengan arus lembut yang membawa pengunjung mengelilingi zona air sambil berbaring di atas ban pelampung. Melewati air terjun kecil dan terowongan berkabut.",
		Emoji:       "🏞️", ImageURL: img("1660812584289-4e346aba6083"), Price: 45000, DurationMin: 30, MinHeightCm: 0, ThrillLevel: 1, DailyQuota: 700,
	},
	{
		Slug: "ember-tumpah-raksasa", Name: "Ember Tumpah Raksasa", Category: domain.CategoryAir,
		Tagline:     "Ember 3.000 liter yang tumpah setiap 3 menit",
		Description: "Area bermain air bertingkat dengan puluhan pancuran, meriam air, dan perosotan pendek. Puncaknya adalah ember raksasa yang tumpah secara berkala dan disambut sorak pengunjung.",
		Emoji:       "🪣", ImageURL: img("1706843540963-ae52d784de62"), Price: 40000, DurationMin: 30, MinHeightCm: 0, ThrillLevel: 2, DailyQuota: 600,
	},

	// ----------------------------------------------------------- Petualangan
	{
		Slug: "flying-fox-danau", Name: "Flying Fox Lintas Danau", Category: domain.CategoryPetualangan,
		Tagline:     "Meluncur 250 meter menyeberangi danau",
		Description: "Meluncur menggunakan tali baja dari menara setinggi 25 meter menyeberangi danau buatan. Menggunakan harness ganda dan sistem rem otomatis yang diperiksa setiap pagi.",
		Emoji:       "🪂", ImageURL: img("1648853070657-6d58398bee93"), Price: 70000, DurationMin: 3, MinHeightCm: 120, ThrillLevel: 4, DailyQuota: 250,
	},
	{
		Slug: "jembatan-gantung-sky-bridge", Name: "Jembatan Gantung Sky Bridge", Category: domain.CategoryPetualangan,
		Tagline:     "Menyeberang jembatan kaca setinggi 20 meter",
		Description: "Jembatan gantung sepanjang 120 meter dengan segmen berlantai kaca transparan di bagian tengah. Memberi pemandangan lembah buatan dan air terjun mini di bawahnya.",
		Emoji:       "🌉", ImageURL: img("1531204709756-1c7a41bf8936"), Price: 50000, DurationMin: 10, MinHeightCm: 110, ThrillLevel: 3, DailyQuota: 300,
	},
	{
		Slug: "panjat-tebing-buatan", Name: "Panjat Tebing Buatan", Category: domain.CategoryPetualangan,
		Tagline:     "Enam jalur panjat dari pemula sampai mahir",
		Description: "Dinding panjat setinggi 12 meter dengan enam jalur berbeda tingkat kesulitan. Menggunakan sistem auto belay sehingga pemanjat dapat turun perlahan secara otomatis.",
		Emoji:       "🧗", ImageURL: img("1570346676483-b629d5188865"), Price: 55000, DurationMin: 15, MinHeightCm: 110, ThrillLevel: 3, DailyQuota: 200,
	},
	{
		Slug: "lintasan-atv", Name: "Lintasan ATV Off Road", Category: domain.CategoryPetualangan,
		Tagline:     "Mengendarai ATV di sirkuit tanah 1,2 kilometer",
		Description: "Sirkuit tanah dengan tanjakan, kubangan lumpur, dan tikungan tajam. Peserta mendapat helm, pelindung lutut, serta pengarahan singkat sebelum memulai putaran.",
		Emoji:       "🏍️", ImageURL: img("1584660042073-f22ae92b1c05"), Price: 90000, DurationMin: 15, MinHeightCm: 130, ThrillLevel: 4, DailyQuota: 150,
	},
	{
		Slug: "arena-paintball", Name: "Arena Paintball", Category: domain.CategoryPetualangan,
		Tagline:     "Pertempuran tim di arena rintangan outdoor",
		Description: "Arena outdoor seluas 1.500 meter persegi dengan bunker, ban bekas, dan menara pengintai. Satu sesi terdiri dari tiga ronde permainan, sudah termasuk 100 peluru cat per peserta.",
		Emoji:       "🔫", ImageURL: img("1663173743747-c9b6235060f1"), Price: 85000, DurationMin: 30, MinHeightCm: 130, ThrillLevel: 3, DailyQuota: 120,
	},
	{
		Slug: "taman-tali-tinggi", Name: "Taman Tali Tinggi", Category: domain.CategoryPetualangan,
		Tagline:     "Dua puluh rintangan tali di antara pepohonan",
		Description: "Jalur petualangan tali di ketinggian 6 sampai 10 meter dengan 20 rintangan seperti jembatan tali, papan titian, dan jaring panjat. Seluruh jalur menggunakan pengaman menerus.",
		Emoji:       "🪢", ImageURL: img("1664735094820-c6c40d862d5b"), Price: 60000, DurationMin: 20, MinHeightCm: 120, ThrillLevel: 3, DailyQuota: 200,
	},

	// ----------------------------------------------------------------- Indoor
	{
		Slug: "rumah-hantu-nusantara", Name: "Rumah Hantu Nusantara", Category: domain.CategoryIndoor,
		Tagline:     "Sembilan ruang bertema legenda hantu Nusantara",
		Description: "Rumah hantu berjalan kaki dengan sembilan ruang bertema cerita rakyat dari berbagai daerah. Menggunakan aktor langsung, tata suara empat arah, dan efek kabut. Tidak disarankan untuk anak di bawah 10 tahun.",
		Emoji:       "👻", ImageURL: img("1511882150382-421056c89033"), Price: 55000, DurationMin: 12, MinHeightCm: 100, ThrillLevel: 4, DailyQuota: 400,
	},
	{
		Slug: "bioskop-4d-petualangan", Name: "Bioskop 4D Petualangan", Category: domain.CategoryIndoor,
		Tagline:     "Film 4D dengan kursi bergerak dan efek air",
		Description: "Studio berkapasitas 80 kursi dengan gerakan enam sumbu, semburan angin, percikan air, dan aroma sesuai adegan. Film diputar bergantian setiap 20 menit dengan tiga judul berbeda.",
		Emoji:       "🎬", ImageURL: img("1523843268911-45a882919fec"), Price: 50000, DurationMin: 15, MinHeightCm: 0, ThrillLevel: 2, DailyQuota: 500,
	},
	{
		Slug: "zona-realitas-virtual", Name: "Zona Realitas Virtual", Category: domain.CategoryIndoor,
		Tagline:     "Delapan pengalaman VR berjalan bebas",
		Description: "Ruang VR seluas 200 meter persegi tempat pengunjung bergerak bebas menggunakan headset nirkabel. Tersedia delapan pengalaman mulai dari menyelam bersama paus sampai bertahan di stasiun luar angkasa.",
		Emoji:       "🥽", ImageURL: img("1511512578047-dfb367046420"), Price: 65000, DurationMin: 15, MinHeightCm: 100, ThrillLevel: 3, DailyQuota: 300,
	},
	{
		Slug: "arena-arcade-game", Name: "Arena Arcade dan Game", Category: domain.CategoryIndoor,
		Tagline:     "Seratus mesin arcade klasik dan modern",
		Description: "Arena permainan berpendingin dengan mesin arcade klasik, simulator balap, mesin basket, dan meja air hockey. Tiket sudah termasuk token untuk 15 kali permainan.",
		Emoji:       "🕹️", ImageURL: img("1558271697-dd9f331ca8b3"), Price: 40000, DurationMin: 30, MinHeightCm: 0, ThrillLevel: 1, DailyQuota: 600,
	},
	{
		Slug: "planetarium-mini", Name: "Planetarium Mini", Category: domain.CategoryIndoor,
		Tagline:     "Kubah bintang dengan proyektor resolusi tinggi",
		Description: "Kubah berdiameter 12 meter yang memproyeksikan langit malam beserta rasi bintang. Pertunjukan dipandu narator dan membahas tata surya serta navigasi bintang tradisional pelaut Nusantara.",
		Emoji:       "🔭", ImageURL: img("1727034393564-dc7b0275686d"), Price: 35000, DurationMin: 25, MinHeightCm: 0, ThrillLevel: 1, DailyQuota: 350,
	},
}
