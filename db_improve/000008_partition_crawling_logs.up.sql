-- =============================================================================
-- 000008_partition_crawling_logs.up.sql
--
-- Tujuan: Convert crawling_logs ke partitioned table RANGE (created_at) per
-- bulan, dengan partisi dinamis berdasarkan data aktual.
--
-- Perubahan kunci:
--   - PK dan unique constraint task_id sekarang termasuk created_at, karena
--     postgresql mensyaratkan partition key ada di unique/PK.
--   - Index lama idx_crawling_logs_target_id diganti jadi
--     idx_crawling_logs_target_id_created agar tidak bentrok dengan index
--     yang dibuat di migration 000006 pada tabel lama.
--   - Partisi dibuat dinamis dari min(created_at) sampai max(created_at)+3
--     bulan, sehingga tidak hardcode range tanggal.
-- =============================================================================

BEGIN;

-- 1. Backup dulu (rename, bukan drop)
ALTER TABLE public.crawling_logs RENAME TO crawling_logs_unpartitioned;

ALTER TABLE public.crawling_logs_unpartitioned
  DROP CONSTRAINT IF EXISTS crawling_logs_pkey;
ALTER TABLE public.crawling_logs_unpartitioned
  DROP CONSTRAINT IF EXISTS crawling_logs_task_id_key;

-- 2. Buat tabel partitioned baru
CREATE TABLE public.crawling_logs (
  id uuid DEFAULT gen_random_uuid() NOT NULL,
  target_id uuid,
  task_id character varying(100) NOT NULL,
  status character varying(50) NOT NULL,
  http_status_code integer,
  error_message text,
  execution_time_ms integer,
  content_hash character varying(64),
  created_at timestamp with time zone DEFAULT now(),
  updated_at timestamp with time zone DEFAULT now()
) PARTITION BY RANGE (created_at);

-- 3. Buat partisi dinamis dari data aktual
DO $$
DECLARE
  d DATE;
  d_end DATE;
  part TEXT;
BEGIN
  SELECT date_trunc('month', min(created_at))::date,
         date_trunc('month', max(created_at))::date + interval '3 months'
  INTO d, d_end
  FROM public.crawling_logs_unpartitioned;

  IF d IS NULL THEN
    RAISE EXCEPTION 'crawling_logs_unpartitioned kosong atau created_at NULL';
  END IF;

  WHILE d <= d_end LOOP
    part := 'crawling_logs_' || to_char(d, 'YYYY_MM');
    EXECUTE format(
      'CREATE TABLE IF NOT EXISTS public.%I PARTITION OF public.crawling_logs FOR VALUES FROM (%L) TO (%L)',
      part, d, d + interval '1 month'
    );
    d := d + interval '1 month';
  END LOOP;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relname = 'crawling_logs_default' AND n.nspname = 'public'
  ) THEN
    CREATE TABLE public.crawling_logs_default PARTITION OF public.crawling_logs DEFAULT;
  END IF;

  RAISE NOTICE 'Partisi dibuat dari % sampai %', d, d_end;
END $$;

-- 4. Constraints dan index di tabel partitioned
ALTER TABLE public.crawling_logs
  ADD CONSTRAINT crawling_logs_pkey PRIMARY KEY (id, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_crawling_logs_task_id
  ON public.crawling_logs (task_id, created_at);

CREATE INDEX IF NOT EXISTS idx_crawling_logs_target_id_created
  ON public.crawling_logs (target_id, created_at);

CREATE INDEX IF NOT EXISTS idx_crawling_logs_status_created
  ON public.crawling_logs (status, created_at);

ALTER TABLE public.crawling_logs
  ADD CONSTRAINT crawling_logs_target_id_fkey
  FOREIGN KEY (target_id) REFERENCES public.crawling_targets(id)
  ON DELETE SET NULL;

-- 5. Migrasi data
INSERT INTO public.crawling_logs (
  id, target_id, task_id, status, http_status_code, error_message,
  execution_time_ms, content_hash, created_at, updated_at
)
SELECT
  id, target_id, task_id, status, http_status_code, error_message,
  execution_time_ms, content_hash, created_at, updated_at
FROM public.crawling_logs_unpartitioned;

DO $$
DECLARE
  v_old BIGINT;
  v_new BIGINT;
BEGIN
  SELECT count(*) INTO v_old FROM public.crawling_logs_unpartitioned;
  SELECT count(*) INTO v_new FROM public.crawling_logs;
  IF v_old <> v_new THEN
    RAISE EXCEPTION 'Jumlah baris tidak sama! old=%, new=%', v_old, v_new;
  END IF;
  RAISE NOTICE 'Migrasi % baris berhasil.', v_new;
END $$;

-- 6. Drop tabel lama
DROP TABLE public.crawling_logs_unpartitioned;

-- 7. Retensi partisi > 6 bulan (jalankan via cron tiap tanggal 1)
CREATE OR REPLACE FUNCTION public.drop_old_crawling_log_partitions(
  retain_months INTEGER DEFAULT 6
)
RETURNS void LANGUAGE plpgsql AS $$
DECLARE
  partition_name TEXT;
  cutoff_date DATE := date_trunc('month', now()) - (retain_months || ' months')::INTERVAL;
  cutoff_suffix TEXT := to_char(cutoff_date, 'YYYY_MM');
BEGIN
  FOR partition_name IN
    SELECT child.relname
    FROM pg_inherits
    JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
    JOIN pg_class child ON pg_inherits.inhrelid = child.oid
    WHERE parent.relname = 'crawling_logs'
      AND child.relname LIKE 'crawling_logs_20%'
      AND child.relname < 'crawling_logs_' || cutoff_suffix
  LOOP
    RAISE NOTICE 'Dropping partition: %', partition_name;
    EXECUTE 'DROP TABLE IF EXISTS public.' || quote_ident(partition_name);
  END LOOP;
END;
$$;

COMMIT;
