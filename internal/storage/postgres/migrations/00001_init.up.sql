BEGIN;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login VARCHAR(200) UNIQUE NOT NULL,
    hash_password VARCHAR(200) NOT NULL,
    balance REAL,
    withdrawn REAL
);

CREATE TYPE status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');

CREATE TABLE orders (
    id bigint PRIMARY KEY UNIQUE,
    status status NOT NULL ,
    created_at TIMESTAMP  DEFAULT now(),
    accrual REAL,
    username VARCHAR(200),
    CONSTRAINT counter_positive_check CHECK (accrual::numeric >= 0),
    FOREIGN KEY(username) references users(login)
);

COMMIT;
