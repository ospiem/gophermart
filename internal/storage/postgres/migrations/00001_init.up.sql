BEGIN;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login VARCHAR(200) UNIQUE NOT NULL,
    hash_password VARCHAR(200) NOT NULL,
    balance REAL,
    withdraw UUID
);


CREATE TABLE orders (
    id VARCHAR(80) PRIMARY KEY UNIQUE,
    status VARCHAR(80) NOT NULL ,
    created_at TIMESTAMP  DEFAULT now(),
    accrual REAL,
    username VARCHAR(200),
    CONSTRAINT accrual_positive_check CHECK (accrual::numeric >= 0),
    FOREIGN KEY(username) REFERENCES users(login)
);

CREATE TABLE withdraws (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(200),
    withdrawn REAL,
    order_number VARCHAR(80),
    processed_at TIMESTAMP  DEFAULT now(),
    FOREIGN KEY(username) REFERENCES users(login),
    FOREIGN KEY(order_number) REFERENCES orders(id)
);

COMMIT;


ALTER TABLE users ADD FOREIGN KEY (withdraw) REFERENCES withdraws(id);

INSERT INTO withdraws (username, withdrawn, order_number) VALUES ('user1', 500, 1212)
RETURNING withdraws.id;