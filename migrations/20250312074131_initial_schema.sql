-- Create "coupon" table
CREATE TABLE "coupon" (
  "code" character varying(255) NOT NULL,
  "influencer_username" character varying(255) NOT NULL,
  "quota" smallint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("code")
);
-- Create "plan" table
CREATE TABLE "plan" (
  "id" uuid NOT NULL,
  "type" character varying(255) NOT NULL,
  "name" character varying(255) NOT NULL,
  "price" integer NOT NULL,
  "duration_in_months" smallint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "plan_name_duration_in_months_key" UNIQUE ("name", "duration_in_months"),
  CONSTRAINT "plan_name_check" CHECK ((name)::text = 'premium'::text)
);
-- Create "user" table
CREATE TABLE "user" (
  "id" uuid NOT NULL,
  "email" character varying(255) NOT NULL,
  "password" character varying(255) NOT NULL,
  "name" character varying(255) NOT NULL,
  "coordinates" point NOT NULL,
  "city" character varying(255) NOT NULL,
  "timezone" character varying(255) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "user_email_key" UNIQUE ("email")
);
-- Create "invoice" table
CREATE TABLE "invoice" (
  "id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "plan_id" uuid NOT NULL,
  "ref_id" character varying(255) NOT NULL,
  "coupon_code" character varying(255) NULL,
  "total_amount" integer NOT NULL,
  "qr_url" character varying(255) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_invoice_coupon_code" FOREIGN KEY ("coupon_code") REFERENCES "coupon" ("code") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "fk_invoice_plan_id" FOREIGN KEY ("plan_id") REFERENCES "plan" ("id") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "fk_invoice_user_id" FOREIGN KEY ("user_id") REFERENCES "user" ("id") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "invoice_total_amount_check" CHECK (total_amount >= 0)
);
-- Create "payment" table
CREATE TABLE "payment" (
  "id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "invoice_id" uuid NOT NULL,
  "amount_paid" integer NOT NULL,
  "status" character varying(16) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "payment_invoice_id_key" UNIQUE ("invoice_id"),
  CONSTRAINT "fk_payment_invoice_id" FOREIGN KEY ("invoice_id") REFERENCES "invoice" ("id") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "fk_payment_user_id" FOREIGN KEY ("user_id") REFERENCES "user" ("id") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "payment_amount_paid_check" CHECK (amount_paid >= 0),
  CONSTRAINT "payment_status_check" CHECK ((status)::text = ANY ((ARRAY['paid'::character varying, 'expired'::character varying, 'failed'::character varying, 'refund'::character varying])::text[]))
);
-- Create "prayer" table
CREATE TABLE "prayer" (
  "id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "name" character varying(16) NOT NULL,
  "status" character varying(16) NOT NULL DEFAULT 'pending',
  "year" smallint NOT NULL,
  "month" smallint NOT NULL,
  "day" smallint NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "prayer_user_id_name_year_month_day_key" UNIQUE ("user_id", "name", "year", "month", "day"),
  CONSTRAINT "fk_prayer_user_id" FOREIGN KEY ("user_id") REFERENCES "user" ("id") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "prayer_name_check" CHECK ((name)::text = ANY ((ARRAY['subuh'::character varying, 'zuhur'::character varying, 'asar'::character varying, 'magrib'::character varying, 'isya'::character varying])::text[])),
  CONSTRAINT "prayer_status_check" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'on_time'::character varying, 'late'::character varying, 'missed'::character varying])::text[]))
);
-- Create "refresh_token" table
CREATE TABLE "refresh_token" (
  "id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "revoked" boolean NOT NULL DEFAULT false,
  "expires_at" timestamptz NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_refresh_token_user_id" FOREIGN KEY ("user_id") REFERENCES "user" ("id") ON UPDATE CASCADE ON DELETE CASCADE
);
-- Create "subscription" table
CREATE TABLE "subscription" (
  "id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "plan_id" uuid NOT NULL,
  "payment_id" uuid NOT NULL,
  "start_date" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "end_date" timestamptz NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_payment_id_key" UNIQUE ("payment_id"),
  CONSTRAINT "fk_subscription_payment_id" FOREIGN KEY ("payment_id") REFERENCES "payment" ("id") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "fk_subscription_plan_id" FOREIGN KEY ("plan_id") REFERENCES "plan" ("id") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "fk_subscription_user_id" FOREIGN KEY ("user_id") REFERENCES "user" ("id") ON UPDATE CASCADE ON DELETE CASCADE
);
