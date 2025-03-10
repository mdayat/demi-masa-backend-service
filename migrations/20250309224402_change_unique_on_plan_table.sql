-- Modify "plan" table
ALTER TABLE "plan" DROP CONSTRAINT "plan_name_key", ADD CONSTRAINT "plan_name_duration_in_months_key" UNIQUE ("name", "duration_in_months");
