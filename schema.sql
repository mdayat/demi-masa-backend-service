CREATE TABLE "user" (
  id UUID PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  password VARCHAR(255) NOT NULL,
  name VARCHAR(255) NOT NULL,
  coordinates POINT NOT NULL,
  city VARCHAR(255) NOT NULL,
  timezone VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE refresh_token (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  revoked BOOLEAN DEFAULT FALSE NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,

  CONSTRAINT fk_refresh_token_user_id
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE
);

CREATE TABLE prayer (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  name VARCHAR(16) NOT NULL CHECK (name IN ('subuh', 'zuhur', 'asar', 'magrib', 'isya')),
  status VARCHAR(16) DEFAULT 'pending' NOT NULL CHECK (status IN ('pending', 'on_time', 'late', 'missed')),
  year SMALLINT NOT NULL,
  month SMALLINT NOT NULL,
  day SMALLINT NOT NULL,

  UNIQUE (user_id, name, year, month, day),

  CONSTRAINT fk_prayer_user_id
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE
);

CREATE TABLE coupon (
  code VARCHAR(255) PRIMARY KEY,
  influencer_username VARCHAR(255) NOT NULL,
  quota SMALLINT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE plan (
  id UUID PRIMARY KEY,
  type VARCHAR(255) NOT NULL CHECK (type IN ('premium')),
  name VARCHAR(255) NOT NULL,
  price INT NOT NULL,
  duration_in_months SMALLINT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE invoice (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  plan_id UUID NOT NULL,
  ref_id VARCHAR(255) NOT NULL,
  coupon_code VARCHAR(255) NULL,
  total_amount INT NOT NULL CHECK (total_amount >= 0),
  qr_url VARCHAR(255) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,

  CONSTRAINT fk_invoice_user_id
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE,

  CONSTRAINT fk_invoice_plan_id
    FOREIGN KEY (plan_id)
    REFERENCES plan(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE,

  CONSTRAINT fk_invoice_coupon_code
    FOREIGN KEY (coupon_code)
    REFERENCES coupon(code)
    ON UPDATE CASCADE
    ON DELETE CASCADE
);

CREATE TABLE payment (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  invoice_id UUID UNIQUE NOT NULL,
  amount_paid INT NOT NULL CHECK (amount_paid >= 0),
  status VARCHAR(16) NOT NULL CHECK (status IN ('paid', 'expired', 'failed', 'refund')),
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,

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

CREATE TABLE subscription (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  plan_id UUID NOT NULL,
  payment_id UUID UNIQUE NOT NULL,
  start_date TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  end_date TIMESTAMPTZ NOT NULL,

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

CREATE TABLE task (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT NOT NULL,
  checked BOOLEAN DEFAULT FALSE NOT NULL,

  CONSTRAINT fk_task_user_id
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE
);