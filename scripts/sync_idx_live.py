#!/usr/bin/env python3
"""
Official Live IDX (BEI) Company Ingestion Script
Fetches 100% real live listed company data from Bursa Efek Indonesia API
and upserts into PostgreSQL master companies using normalized matching.
"""

import sys
import os
import json
import re
import cloudscraper
import psycopg2
from psycopg2.extras import RealDictCursor

DB_URI = os.getenv("DATABASE_URL", "postgres://sovera:sover4@10.10.29.177:5432/sovera")

def normalize_company_name(nama_emiten, kode_emiten):
    name = nama_emiten.strip().rstrip(".")
    if not re.search(r'\bTbk\b', name, re.IGNORECASE):
        name = name + " Tbk"
    if not re.match(r'^(PT|PT\.|PERSERO)\b', name, re.IGNORECASE):
        name = "PT " + name
    return name

def fetch_live_idx_profiles():
    print("Fetching live emiten data from official BEI / IDX API...")
    scraper = cloudscraper.create_scraper(delay=10)
    url = "https://www.idx.co.id/primary/ListedCompany/GetCompanyProfiles?emitenType=s&start=0&length=1000"
    resp = scraper.get(url, timeout=30)
    if resp.status_code != 200:
        raise Exception(f"IDX API returned HTTP {resp.status_code}")
    
    payload = resp.json()
    items = payload.get("data", [])
    print(f"Successfully retrieved {len(items)} real listed companies from IDX API.")
    return items

def sync_to_database(items):
    print("Connecting to PostgreSQL database...")
    conn = psycopg2.connect(DB_URI)
    conn.autocommit = False
    cur = conn.cursor(cursor_factory=RealDictCursor)

    created_count = 0
    updated_count = 0
    skipped_count = 0

    for item in items:
        ticker = (item.get("KodeEmiten") or "").strip().upper()
        raw_name = (item.get("NamaEmiten") or "").strip()

        if not ticker or not raw_name:
            continue

        name = normalize_company_name(raw_name, ticker)
        legal_name = name
        slug = ticker.lower()

        website = (item.get("Website") or "").strip()
        if website and not website.startswith("http://") and not website.startswith("https://"):
            website = "https://" + website
        elif not website:
            website = None

        sektor = (item.get("Sektor") or "").strip()
        if not sektor:
            sektor = (item.get("SubSektor") or "").strip()
        if not sektor:
            sektor = "Emiten BEI / Pasar Modal"

        alamat = (item.get("Alamat") or "").strip() or None
        alias_keywords = [name, ticker, raw_name]

        # 1. Try finding existing company by ticker OR normalized alphanumeric name OR slug
        find_query = """
            SELECT id::text FROM company.companies
            WHERE ticker = %s 
               OR LOWER(REGEXP_REPLACE(name, '[^a-zA-Z0-9]', '', 'g')) = LOWER(REGEXP_REPLACE(%s, '[^a-zA-Z0-9]', '', 'g'))
               OR slug = %s
            LIMIT 1;
        """
        cur.execute(find_query, (ticker, name, slug))
        found = cur.fetchone()

        if found:
            comp_id = found["id"]
            update_query = """
                UPDATE company.companies SET
                    ticker = %s,
                    name = %s,
                    legal_name = %s,
                    industry_sector = %s,
                    website = COALESCE(%s, website),
                    headquarters = COALESCE(%s, headquarters),
                    company_type = 'SWASTA_TBK',
                    is_public = TRUE,
                    priority_tier = 'TIER_1',
                    updated_at = NOW()
                WHERE id::text = %s;
            """
            cur.execute(update_query, (ticker, name, legal_name, sektor, website, alamat, comp_id))
            conn.commit()
            updated_count += 1
        else:
            insert_query = """
                INSERT INTO company.companies (
                    name, legal_name, slug, industry_sector, company_type, website, headquarters, ticker, is_public, priority_tier, alias_keywords, created_at, updated_at
                ) VALUES (
                    %s, %s, %s, %s, 'SWASTA_TBK', %s, %s, %s, TRUE, 'TIER_1', %s, NOW(), NOW()
                );
            """
            try:
                cur.execute(insert_query, (name, legal_name, slug, sektor, website, alamat, ticker, alias_keywords))
                conn.commit()
                created_count += 1
            except Exception as e:
                conn.rollback()
                print(f"Notice: skipped insert for emiten {ticker} ({name}): {e}")
                skipped_count += 1
                continue

    cur.close()
    conn.close()

    print(f"Sync Complete! Added {created_count} new emiten, updated {updated_count} existing emiten. Skipped: {skipped_count}.")

if __name__ == "__main__":
    try:
        data = fetch_live_idx_profiles()
        sync_to_database(data)
    except Exception as e:
        print(f"Error during IDX live sync: {e}", file=sys.stderr)
        sys.exit(1)
