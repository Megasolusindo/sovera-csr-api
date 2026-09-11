-- Migration 000022: Seed Official Indonesian Humanitarian Institutions & Prospective LAZ/NGO Users

-- 1. Seed 25 Official Humanitarian Institutions & LAZ into organizations table
INSERT INTO organizations (id, name, org_type, subscription_tier) VALUES
('b1000000-0000-4000-a000-000000000001', 'Rumah Zakat Indonesia', 'ZAKAT_WAQF_INSTITUTION', 'PRO'),
('b1000000-0000-4000-a000-000000000002', 'Dompet Dhuafa Philanthropy', 'ZAKAT_WAQF_INSTITUTION', 'ENTERPRISE'),
('b1000000-0000-4000-a000-000000000003', 'Human Initiative (PKPU)', 'HUMANITARIAN_NGO', 'PRO'),
('b1000000-0000-4000-a000-000000000004', 'LAZISMU (Muhammadiyah)', 'ZAKAT_WAQF_INSTITUTION', 'ENTERPRISE'),
('b1000000-0000-4000-a000-000000000005', 'NU Care - LAZISNU', 'ZAKAT_WAQF_INSTITUTION', 'ENTERPRISE'),
('b1000000-0000-4000-a000-000000000006', 'Inisiatif Zakat Indonesia (IZI)', 'ZAKAT_WAQF_INSTITUTION', 'PRO'),
('b1000000-0000-4000-a000-000000000007', 'LAZNAS LMI (Lembaga Manajemen Infaq)', 'ZAKAT_WAQF_INSTITUTION', 'FREE_TRIAL'),
('b1000000-0000-4000-a000-000000000008', 'BSI Maslahat (LAZ BSM Umat)', 'ZAKAT_WAQF_INSTITUTION', 'ENTERPRISE'),
('b1000000-0000-4000-a000-000000000009', 'Yatim Mandiri', 'ZAKAT_WAQF_INSTITUTION', 'PRO'),
('b1000000-0000-4000-a000-000000000010', 'DT Peduli (Darut Tauhiid)', 'ZAKAT_WAQF_INSTITUTION', 'PRO'),
('b1000000-0000-4000-a000-000000000011', 'Baitul Maal Hidayatullah (BMH)', 'ZAKAT_WAQF_INSTITUTION', 'PRO'),
('b1000000-0000-4000-a000-000000000012', 'LAZ Al-Azhar', 'ZAKAT_WAQF_INSTITUTION', 'PRO'),
('b1000000-0000-4000-a000-000000000013', 'Yayasan Benih Baik Indonesia (BenihBaik.com)', 'CSR_IMPLEMENTER', 'ENTERPRISE'),
('b1000000-0000-4000-a000-000000000014', 'Yayasan Kitabisa (Kitabisa B2B CSR)', 'CSR_IMPLEMENTER', 'ENTERPRISE'),
('b1000000-0000-4000-a000-000000000015', 'Palang Merah Indonesia (PMI Pusat)', 'HUMANITARIAN_NGO', 'ENTERPRISE'),
('b1000000-0000-4000-a000-000000000016', 'Yayasan Habitat for Humanity Indonesia', 'HUMANITARIAN_NGO', 'PRO'),
('b1000000-0000-4000-a000-000000000017', 'Wahana Visi Indonesia (WVI)', 'HUMANITARIAN_NGO', 'ENTERPRISE'),
('b1000000-0000-4000-a000-000000000018', 'Yayasan Plan International Indonesia', 'HUMANITARIAN_NGO', 'PRO'),
('b1000000-0000-4000-a000-000000000019', 'WWF Indonesia Foundation', 'COMMUNITY_ORG', 'PRO'),
('b1000000-0000-4000-a000-000000000020', 'Yayasan Kehati (Keanekaragaman Hayati)', 'FOUNDATION', 'ENTERPRISE'),
('b1000000-0000-4000-a000-000000000021', 'LAZ Hadji Kalla', 'CSR_IMPLEMENTER', 'PRO'),
('b1000000-0000-4000-a000-000000000022', 'LAZ Solo Peduli', 'ZAKAT_WAQF_INSTITUTION', 'FREE_TRIAL'),
('b1000000-0000-4000-a000-000000000023', 'Mizan Amanah', 'ZAKAT_WAQF_INSTITUTION', 'PRO'),
('b1000000-0000-4000-a000-000000000024', 'Griya Yatim & Dhuafa', 'ZAKAT_WAQF_INSTITUTION', 'FREE_TRIAL'),
('b1000000-0000-4000-a000-000000000025', 'YAKKUM Emergency Unit (YEU Indonesia)', 'HUMANITARIAN_NGO', 'PRO')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, org_type = EXCLUDED.org_type;

-- 2. Seed organization_profiles for coverage scope & CSR partnership URLs
INSERT INTO organization_profiles (id, org_id, description, coverage_scope, has_csr_partnerships, has_corporate_partnership, has_grant_program, partnerships_page_url, confidence) VALUES
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000001', 'Lembaga Amil Zakat Nasional dan pemberdayaan masyarakat terkemuka di Indonesia.', 'NATIONAL', true, true, true, 'https://www.rumahzakat.org/kemitraan-csr', 0.98),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000002', 'Lembaga filantropi Islam pemberdayaan kaum dhuafa dan mitra CSR korporasi global.', 'NATIONAL', true, true, true, 'https://www.dompetdhuafa.org/csr-partner', 0.99),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000003', 'Organisasi kemanusiaan dunia dengan fokus bencana, pendidikan, dan kesehatan.', 'INTERNATIONAL', true, true, true, 'https://human-initiative.org/partner-with-us', 0.95),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000004', 'Lembaga amil zakat nasional Muhammadiyah dengan jaringan cabang se-Indonesia.', 'NATIONAL', true, true, true, 'https://lazismu.org/kemitraan', 0.97),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000005', 'Lembaga amil zakat infaq dan shadaqah Nahdlatul Ulama.', 'NATIONAL', true, true, true, 'https://nucare.id', 0.96),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000013', 'Platform fasilitator program CSR B2B korporasi dan aksi kemanusiaan berdampak.', 'NATIONAL', true, true, true, 'https://benihbaik.com/b2b-csr', 0.99),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000014', 'Divisi solusi CSR korporasi Yayasan Kitabisa Indonesia.', 'NATIONAL', true, true, true, 'https://kitabisa.com/b2b', 0.98),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000015', 'Perhimpunan nasional bidang kemanusiaan & penanggulangan bencana Indonesia.', 'NATIONAL', true, true, true, 'https://pmi.or.id', 0.99)
ON CONFLICT (org_id) DO NOTHING;

-- 3. Seed organization_prospects CRM leads for Sales Outreach
INSERT INTO organization_prospects (id, org_id, sales_status, lead_score, source) VALUES
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000001', 'QUALIFIED', 95.00, 'KEMENAG_REGISTRY'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000002', 'PROSPECT', 98.00, 'BAZNAS_INDEX'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000003', 'QUALIFIED', 92.00, 'FORUM_ZAKAT'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000004', 'LEAD', 96.00, 'BAZNAS_INDEX'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000005', 'LEAD', 96.00, 'BAZNAS_INDEX'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000013', 'QUALIFIED', 97.00, 'FILANTROPI_ID'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000014', 'QUALIFIED', 97.00, 'FILANTROPI_ID'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000015', 'PROSPECT', 95.00, 'PMI_DIRECTORY')
ON CONFLICT (org_id) DO NOTHING;
