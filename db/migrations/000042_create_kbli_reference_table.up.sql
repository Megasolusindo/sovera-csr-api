-- Migration 000042: Create KBLI (Klasifikasi Baku Lapangan Usaha Indonesia) Reference Table and Seed Official Dataset
CREATE TABLE IF NOT EXISTS public.kbli_reference (
    code VARCHAR(10) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    category_code CHAR(1) NOT NULL,
    category_title VARCHAR(255) NOT NULL,
    csr_relevance_default VARCHAR(50) DEFAULT 'MEDIUM',
    risk_level VARCHAR(50) DEFAULT 'MENENGAH',
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_kbli_category ON public.kbli_reference(category_code);
CREATE INDEX IF NOT EXISTS idx_kbli_csr_relevance ON public.kbli_reference(csr_relevance_default);

-- Add Foreign Key on company.companies(kbli_code) referencing kbli_reference(code)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'companies_kbli_code_fkey'
    ) THEN
        ALTER TABLE company.companies
        ADD CONSTRAINT companies_kbli_code_fkey
        FOREIGN KEY (kbli_code) REFERENCES public.kbli_reference(code)
        ON DELETE SET NULL;
    END IF;
END $$;

-- Seed Official BPS / OSS 2020 Standard KBLI Dataset
INSERT INTO public.kbli_reference (code, title, category_code, category_title, csr_relevance_default, risk_level, description) VALUES
('06100', 'Pertambangan Minyak Bumi', 'B', 'Pertambangan dan Penggalian', 'HIGH', 'TINGGI', 'Kegiatan eksplorasi, ekstraksi, dan penambangan minyak bumi mentah.'),
('06200', 'Pertambangan Gas Alam', 'B', 'Pertambangan dan Penggalian', 'HIGH', 'TINGGI', 'Kegiatan ekstraksi dan produksi gas alam cair dan kompresi.'),
('05100', 'Pertambangan Batu Bara', 'B', 'Pertambangan dan Penggalian', 'HIGH', 'TINGGI', 'Kegiatan penambangan dan pembersihan batu bara batuan padat.'),
('07100', 'Pertambangan Bijih Besi', 'B', 'Pertambangan dan Penggalian', 'HIGH', 'TINGGI', 'Penambangan dan ekstraksi bijih besi mentah.'),
('07291', 'Pertambangan Bijih Nikel', 'B', 'Pertambangan dan Penggalian', 'HIGH', 'TINGGI', 'Penambangan bijih nikel untuk industri baterai dan smelter baja.'),
('07292', 'Pertambangan Bijih Tembaga', 'B', 'Pertambangan dan Penggalian', 'HIGH', 'TINGGI', 'Penambangan dan konsentrat bijih tembaga.'),
('07293', 'Pertambangan Bijih Emas dan Perak', 'B', 'Pertambangan dan Penggalian', 'HIGH', 'TINGGI', 'Penambangan dan pemurnian emas dan logam mulia perak.'),

('35101', 'Pembangkitan Tenaga Listrik', 'D', 'Pengadaan Listrik, Gas, Uap/Air Panas Dan Udara Dingin', 'HIGH', 'TINGGI', 'Pengoperasian fasilitas pembangkitan tenaga listrik thermal, hidro, EBT, dan geothermal.'),
('35102', 'Transmisi Tenaga Listrik', 'D', 'Pengadaan Listrik, Gas, Uap/Air Panas Dan Udara Dingin', 'HIGH', 'MENENGAH_TINGGI', 'Pengoperasian sistem jaringan transmisi listrik tegangan tinggi.'),
('35103', 'Distribusi Tenaga Listrik', 'D', 'Pengadaan Listrik, Gas, Uap/Air Panas Dan Udara Dingin', 'HIGH', 'MENENGAH_TINGGI', 'Penjualan dan distribusi energi listrik ke konsumen akhir.'),
('35201', 'Pengadaan Gas Melalui Pipa', 'D', 'Pengadaan Listrik, Gas, Uap/Air Panas Dan Udara Dingin', 'HIGH', 'TINGGI', 'Distribusi dan niaga gas bumi melalui jaringan pipa komersial.'),

('19201', 'Industri Kilang Minyak Bumi', 'C', 'Industri Pengolahan', 'HIGH', 'TINGGI', 'Pengolahan minyak mentah menjadi BBM, LPG, dan petrokimia.'),
('24101', 'Industri Pembuatan Besi dan Baja Dasar', 'C', 'Industri Pengolahan', 'HIGH', 'TINGGI', 'Peleburan dan pembuatan produk baja dasar dan bilet besi.'),
('20111', 'Industri Kimia Dasar Anorganik', 'C', 'Industri Pengolahan', 'HIGH', 'TINGGI', 'Manufaktur bahan kimia dasar seperti pupuk, amonia, dan asam murni.'),
('10771', 'Industri Gula Pasir', 'C', 'Industri Pengolahan', 'MEDIUM', 'MENENGAH_TINGGI', 'Pengolahan tebu menjadi gula pasir murni dan gula industri.'),
('10791', 'Industri Minyak Mentah Kelapa Sawit (CPO)', 'C', 'Industri Pengolahan', 'HIGH', 'TINGGI', 'Pengolahan kelapa sawit menjadi CPO dan produk turunan oleokimia.'),
('11010', 'Industri Minuman Ringan dan Air Minum Dalam Kemasan (AMDK)', 'C', 'Industri Pengolahan', 'HIGH', 'MENENGAH_RENDAH', 'Manufaktur air minum dalam kemasan dan minuman bernutrisi.'),
('29101', 'Industri Mobil dan Kendaraan Bermotor Roda Empat', 'C', 'Industri Pengolahan', 'HIGH', 'MENENGAH_TINGGI', 'Perakitan dan manufaktur kendaraan bermotor dan kendaraan listrik (EV).'),

('64191', 'Perbankan Konvensional', 'K', 'Aktivitas Keuangan dan Asuransi', 'HIGH', 'RENGAH', 'Kegiatan penghimpunan dana dan penyaluran kredit komersial dan UMKM.'),
('64192', 'Perbankan Syariah', 'K', 'Aktivitas Keuangan dan Asuransi', 'HIGH', 'RENGAH', 'Kegiatan perbankan berdasarkan prinsip syariah dan bagi hasil.'),
('65111', 'Asuransi Jiwa', 'K', 'Aktivitas Keuangan dan Asuransi', 'MEDIUM', 'RENGAH', 'Penyediaan perlindungan asuransi jiwa dan tabungan investasi.'),
('65121', 'Asuransi Umum', 'K', 'Aktivitas Keuangan dan Asuransi', 'MEDIUM', 'RENGAH', 'Penyediaan perlindungan kerugian asuransi non-jiwa.'),
('64991', 'Aktivitas Perusahaan Holding', 'K', 'Aktivitas Keuangan dan Asuransi', 'HIGH', 'RENGAH', 'Pengelolaan anak perusahaan dan konsolidasi grup usaha corporate.'),
('64920', 'Aktivitas Perusahaan Pembiayaan (Multifinance)', 'K', 'Aktivitas Keuangan dan Asuransi', 'MEDIUM', 'MENENGAH_RENDAH', 'Penyaluran pembiayaan investasi, modal kerja, dan multifinance.'),

('61100', 'Telekomunikasi Kabel', 'J', 'Informasi dan Komunikasi', 'HIGH', 'MENENGAH_RENDAH', 'Penyediaan jaringan komunikasi serat optik dan broadband internet.'),
('61200', 'Telekomunikasi Nirkabel / Seluler', 'J', 'Informasi dan Komunikasi', 'HIGH', 'MENENGAH_RENDAH', 'Penyediaan layanan telekomunikasi seluler 4G/5G dan data seluler.'),
('62019', 'Aktivitas Pemrograman Komputer Lainnya', 'J', 'Informasi dan Komunikasi', 'MEDIUM', 'RENGAH', 'Pengembangan perangkat lunak, sistem aplikasi enterprise, dan AI.'),
('63120', 'Portal Web dan Platform Digital Marketplaces', 'J', 'Informasi dan Komunikasi', 'MEDIUM', 'RENGAH', 'Pengoperasian portal web berita, e-commerce, dan ekonomi digital.'),

('41011', 'Konstruksi Gedung Hunian', 'F', 'Konstruksi', 'HIGH', 'MENENGAH_TINGGI', 'Pembangunan gedung apartemen, perumahan, dan pemukiman.'),
('41012', 'Konstruksi Gedung Perkantoran dan Komersial', 'F', 'Konstruksi', 'HIGH', 'MENENGAH_TINGGI', 'Pembangunan mall, pusat perbelanjaan, dan gedung perkantoran.'),
('42101', 'Konstruksi Jalan Raya, Jalan Tol, dan Jembatan', 'F', 'Konstruksi', 'HIGH', 'TINGGI', 'Pembangunan infrastruktur jalan tol nasional, jalan raya, dan flyover.'),
('42201', 'Konstruksi Jaringan Listrik dan Telekomunikasi', 'F', 'Konstruksi', 'HIGH', 'MENENGAH_TINGGI', 'Pembangunan menara BTS, jaringan kabel transmisi, dan instalasi gardu.'),
('68111', 'Real Estat / Pengembang Properti', 'L', 'Real Estat', 'HIGH', 'MENENGAH_TINGGI', 'Pengembangan dan penjualan kawasan perumahan, kawasan industri, dan komersial.'),

('51101', 'Angkutan Udara Niaga Berjadwal', 'H', 'Pengangkutan dan Pergudangan', 'HIGH', 'MENENGAH_TINGGI', 'Pengoperasian penerbangan komersial penumpang dan kargo udara.'),
('50111', 'Angkutan Laut Dalam Negeri', 'H', 'Pengangkutan dan Pergudangan', 'HIGH', 'MENENGAH_TINGGI', 'Pengangkutan kapal laut niaga antarpulau dan logistik maritim.'),
('49211', 'Angkutan Bus Antarkota dan Perkotaan', 'H', 'Pengangkutan dan Pergudangan', 'MEDIUM', 'MENENGAH_RENDAH', 'Penyediaan angkutan umum jalan raya bus masal.'),
('52101', 'Pergudangan dan Penyimpanan Logistik', 'H', 'Pengangkutan dan Pergudangan', 'MEDIUM', 'RENGAH', 'Penyediaan fasilitas pergudangan pusat distribusi komoditas.'),

('01261', 'Perkebunan Kelapa Sawit', 'A', 'Pertanian, Kehutanan dan Perikanan', 'HIGH', 'TINGGI', 'Budidaya dan perkebunan tanaman kelapa sawit.'),
('01262', 'Perkebunan Karet', 'A', 'Pertanian, Kehutanan dan Perikanan', 'HIGH', 'MENENGAH_TINGGI', 'Budidaya dan ekstraksi getah tanaman karet.'),
('01271', 'Perkebunan Kopi', 'A', 'Pertanian, Kehutanan dan Perikanan', 'MEDIUM', 'RENGAH', 'Budidaya dan pemanenan biji kopi.'),
('01272', 'Perkebunan Teh', 'A', 'Pertanian, Kehutanan dan Perikanan', 'MEDIUM', 'RENGAH', 'Budidaya dan pemanenan daun teh murni.'),
('02111', 'Kehutanan Hutan Tanaman Industri (HTI)', 'A', 'Pertanian, Kehutanan dan Perikanan', 'HIGH', 'TINGGI', 'Pengelolaan hutan tanaman industri kayu dan bahan baku kertas.'),
('03111', 'Penangkapan Ikan di Laut', 'A', 'Pertanian, Kehutanan dan Perikanan', 'HIGH', 'MENENGAH_TINGGI', 'Penangkapan komoditas ikan dan biota laut komersial.')
ON CONFLICT (code) DO UPDATE SET
    title = EXCLUDED.title,
    category_code = EXCLUDED.category_code,
    category_title = EXCLUDED.category_title,
    csr_relevance_default = EXCLUDED.csr_relevance_default,
    risk_level = EXCLUDED.risk_level,
    description = EXCLUDED.description,
    updated_at = NOW();
