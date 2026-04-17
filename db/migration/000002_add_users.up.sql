CREATE TABLE "users" (
  "username" text PRIMARY KEY,
  "hashed_password" text NOT NULL,
  "full_name" text NOT NULL,
  "email" text UNIQUE NOT NULL,
  "password_changed_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "accounts"
ADD FOREIGN KEY ("owner") REFERENCES "users" ("username") DEFERRABLE INITIALLY IMMEDIATE;

CREATE UNIQUE INDEX "owner_currency_key" ON "accounts" ("owner", "currency");