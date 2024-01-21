BEGIN;

CREATE TABLE gauges (
                       id VARCHAR(200) PRIMARY KEY UNIQUE,
                       gauge DOUBLE PRECISION NOT NULL
);

CREATE TABLE counters (
                         id VARCHAR(200) PRIMARY KEY UNIQUE,
                         counter BIGINT NOT NULL,
                         CONSTRAINT counter_positive_check CHECK (counter::numeric >= 0)
);

COMMIT;
