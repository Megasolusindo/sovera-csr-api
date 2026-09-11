# Strictly Enforced Business Rules

## ZERO DATA MOCKING DIRECTIVE (CRITICAL - CORE BUSINESS RULE)
- **STRICT MANDATE**: Under no circumstances should any mock, fake, dummy, or hardcoded fallback data be generated or inserted for corporate entities, CSR budgets, CSR programs, business contact info, phone numbers, emails, URLs, or tickers.
- **EMPIRICAL VALIDITY ONLY**: Every single record entering the database must be strictly verified against authoritative, real-world sources:
  1. **Corporate Entity & Ticker**: Verified against official IDX / BEI listed profiles or government business registries.
  2. **Website URL**: Verified via live HTTP reachability check (`urlverifier` returning status 2xx/3xx). Rejects invalid or dead links.
  3. **Phone & Email Contacts**: Verified via strict format validation (`phoneverifier` and email regex). Rejects placeholder numbers (e.g., `000000`, `123456`).
  4. **CSR Signals & Budgets**: Extracted strictly from real scraped text and verified news articles via LLM without synthetic or hardcoded fallback names/budgets.
- **NO SILENT DUMMY FALLBACKS**: If an API or extraction service fails (e.g. Gemini LLM rate limits or network error), the pipeline must return an explicit error or log a failure. It MUST NOT fall back to dummy/mock data generation under any circumstance.
