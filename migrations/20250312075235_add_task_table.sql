-- Create "task" table
CREATE TABLE "task" (
  "id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "name" character varying(255) NOT NULL,
  "description" text NOT NULL,
  "checked" boolean NOT NULL DEFAULT false,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_task_user_id" FOREIGN KEY ("user_id") REFERENCES "user" ("id") ON UPDATE CASCADE ON DELETE CASCADE
);
