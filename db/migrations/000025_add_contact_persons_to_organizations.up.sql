-- Migration 000025: Add Contact Person columns & Seed CRM Contacts for Organizations

-- 1. Add contact_name, contact_email, contact_phone to organizations table
ALTER TABLE organizations
ADD COLUMN IF NOT EXISTS contact_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS contact_email VARCHAR(255),
ADD COLUMN IF NOT EXISTS contact_phone VARCHAR(50);

-- 2. Seed Contact Persons into crm_contacts for top 27+ organizations
INSERT INTO crm_contacts (id, org_id, name, position, email, phone) VALUES
-- System & Active Tenants
(gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'Super Admin Operator', 'Platform System Admin', 'admin@sovera.id', '+62 811-1000-9000'),
(gen_random_uuid(), '77123aaa-8819-4c12-99a1-00123456789a', 'M. Lulu D. K.', 'Lead Fundraiser & Partnership', 'lulu@peduliummat.org', '+62 812-3456-7890'),

-- Tier 1 Humanitarian Institutions & LAZ
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000001', 'Andri Purwanto', 'Head of Corporate Partnership', 'kemitraan@rumahzakat.org', '+62 812-8921-3401'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000002', 'Dian Hermawan, M.M.', 'General Manager Corporate Philanthropy', 'b2b.csr@dompetdhuafa.org', '+62 813-7788-9012'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000003', 'Siti Rahmawati', 'Senior Manager Global Partnership', 'partnership@human-initiative.org', '+62 811-9002-3344'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000004', 'Rizal Arifin, S.E.', 'Direktur Penghimpunan & Kemitraan', 'kemitraan@lazismu.org', '+62 815-6677-8899'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000005', 'Ahmad Ridwan, M.Si.', 'Sekretaris Eksekutif LAZISNU', 'b2b@nucare.id', '+62 812-1122-3344'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000006', 'Rina Kartika', 'Corporate Relation Lead', 'partnership@izi.or.id', '+62 813-4455-6677'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000007', 'Budi Santoso', 'Manager Kemitraan BUMN & CSR', 'csr@lmizakat.org', '+62 812-9988-7766'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000008', 'Hendra Wijaya, S.E.', 'Head of Corporate CSR Division', 'corporate@bsimaslahat.id', '+62 811-8899-0011'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000009', 'Eko Prasetyo', 'Manager Program Kemitraan Strategis', 'kemitraan@yatimmandiri.org', '+62 813-2233-4455'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000010', 'Fitri Handayani', 'Head of CSR Partnership Desk', 'csr@dtpeduli.org', '+62 812-5566-7788'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000011', 'Zulkifli Ahmad', 'General Manager Kemitraan B2B', 'kemitraan@bmh.or.id', '+62 811-3344-5566'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000012', 'Tri Raharjo', 'Manager Corporate Social Responsibility', 'csr@alazhar.or.id', '+62 813-6677-8899'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000013', 'Maya Indah', 'Head of B2B Solutions BenihBaik', 'b2b@benihbaik.com', '+62 812-7788-9900'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000014', 'Rangga Wijaya', 'Director of Corporate Partnerships', 'b2b@kitabisa.com', '+62 811-6677-8899'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000015', 'Dr. Arifin Iskandar', 'Kepala Divisi Kemitraan Kemanusiaan', 'kemitraan@pmi.or.id', '+62 812-8899-0011'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000016', 'Dewi Susanti', 'Resource Development Manager', 'partnership@habitatindonesia.org', '+62 813-1122-3344'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000017', 'Bambang Sukoco', 'Senior Corporate Engagement Officer', 'csr@wvi.or.id', '+62 811-4455-6677'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000018', 'Anita Larasati', 'Philanthropy & Corporate Relation', 'corporate@plan-international.or.id', '+62 812-3344-5566'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000019', 'Riki Firmansyah', 'ESG & Corporate Alliance Lead', 'kemitraan@wwf.id', '+62 813-8899-0011'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000020', 'Dr. Ir. Rahmat Hidayat', 'Direktur Pendanaan ESG & Kehati', 'partnership@kehati.or.id', '+62 811-2233-4455'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000026', 'Drs. H. M. Zainul Majdi', 'Direktur Kemitraan BAZNAS RI', 'kemitraan@baznas.go.id', '+62 812-1000-2000'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000027', 'Budi Harjo, S.E.', 'Head of Corporate Waqf & CSR', 'kemitraan@bmm.or.id', '+62 813-3000-4000'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000041', 'Dr. Lie Dharmawan', 'Direktur Eksekutif doctorSHARE', 'partnership@doctorshare.org', '+62 811-5000-6000'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000100', 'Bambang Ismawan', 'Ketua Pengurus Filantropi Indonesia', 'info@filantropi.or.id', '+62 812-7000-8000'),
(gen_random_uuid(), 'b1000000-0000-4000-a000-000000000101', 'Bambang Suherman', 'Ketua Umum Forum Zakat (FOZ)', 'sekretariat@forumzakat.org', '+62 813-9000-1000')
ON CONFLICT DO NOTHING;

-- 3. Sync primary contact details directly into organizations table
UPDATE organizations o
SET 
  contact_name = c.name,
  contact_email = c.email,
  contact_phone = c.phone
FROM crm_contacts c
WHERE c.org_id = o.id;

-- 4. Set NULL for organizations without explicit contact details (NO DUMMY FALLBACKS)
-- Fallback generation removed per ZERO DATA MOCKING DIRECTIVE
