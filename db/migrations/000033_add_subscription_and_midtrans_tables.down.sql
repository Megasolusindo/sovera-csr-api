-- Migration 000033 Down: Rollback Subscription & Midtrans Billing tables

DROP TABLE IF EXISTS webhook_inbound_logs;
DROP TABLE IF EXISTS payment_transactions;
DROP TABLE IF EXISTS billing_invoices;
DROP TABLE IF EXISTS tenant_subscriptions;
DROP TABLE IF EXISTS subscription_plans;
