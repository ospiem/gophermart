BEGIN;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login VARCHAR(200) UNIQUE NOT NULL,
    hash_password VARCHAR(200) NOT NULL,
    balance REAL NULL,
    total_withdrawn REAL NULL
);

CREATE TABLE orders (
    id VARCHAR(80) PRIMARY KEY UNIQUE,
    status VARCHAR(80) NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL ,
    accrual REAL NULL,
    username VARCHAR(200) NOT NULL ,
    withdraw UUID NOT NULL ,
    CONSTRAINT accrual_positive_check CHECK (accrual::numeric >= 0),
    FOREIGN KEY(username) REFERENCES users(login)
);

CREATE TABLE withdraws (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(200) NOT NULL ,
    withdrawn REAL NULL ,
    order_number VARCHAR(80) NOT NULL ,
    processed_at TIMESTAMP DEFAULT now() NOT NULL ,
    FOREIGN KEY(username) REFERENCES users(login),
    FOREIGN KEY(order_number) REFERENCES orders(id)
);

ALTER TABLE orders ADD FOREIGN KEY (withdraw) REFERENCES withdraws(id);

COMMIT;

