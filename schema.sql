CREATE TABLE "user" (
  id VARCHAR(255) PRIMARY KEY,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE refresh_token (
  id UUID PRIMARY KEY,     
  user_id VARCHAR(255) NOT NULL,
  revoked BOOLEAN DEFAULT FALSE NOT NULL,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,

  CONSTRAINT fk_refresh_token_user_id
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE
);

CREATE TABLE invoice (
  id UUID PRIMARY KEY,
  user_id VARCHAR(255) NOT NULL,
  total_amount INT NOT NULL CHECK (total_amount >= 0),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMPTZ NULL,

  CONSTRAINT fk_invoice_user_id
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE
);

CREATE TABLE payment (
  id UUID PRIMARY KEY,
  user_id VARCHAR(255) NOT NULL,
  invoice_id UUID UNIQUE NOT NULL,
  amount_paid INT NOT NULL CHECK (amount_paid >= 0),
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMPTZ NULL,

  CONSTRAINT fk_payment_user_id
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE,

  CONSTRAINT fk_payment_invoice_id
    FOREIGN KEY (invoice_id)
    REFERENCES invoice(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE
);

CREATE TABLE plan (
  id UUID PRIMARY KEY,
  name VARCHAR(100) UNIQUE NOT NULL,
  price INT NOT NULL,
  duration_in_months INT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE subscription (
  id UUID PRIMARY KEY,
  user_id VARCHAR(255) NOT NULL,
  plan_id UUID NOT NULL,
  payment_id UUID UNIQUE NOT NULL,
  start_date TIMESTAMPTZ NOT NULL,
  end_date TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMPTZ NULL,

  CONSTRAINT fk_subscription_user_id
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE,

  CONSTRAINT fk_subscription_plan_id
    FOREIGN KEY (plan_id)
    REFERENCES plan(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE,

  CONSTRAINT fk_subscription_payment_id
    FOREIGN KEY (payment_id)
    REFERENCES payment(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE
);