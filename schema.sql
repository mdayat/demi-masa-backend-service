CREATE TABLE "user" (
  id BIGINT PRIMARY KEY,
  first_name VARCHAR(255) NOT NULL,
  subscription_expired_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMPTZ NULL
);
