-- Modify "plan" table
ALTER TABLE "plan" DROP CONSTRAINT "plan_name_check", DROP CONSTRAINT "plan_name_duration_in_months_key", ADD CONSTRAINT "plan_type_check" CHECK ((type)::text = 'premium'::text);
