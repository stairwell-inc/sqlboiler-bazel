CREATE SCHEMA powerplant;

ALTER SCHEMA powerplant OWNER TO operator;

CREATE TABLE powerplant.generators (
	generator_id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
	name text NOT NULL,
	nickname text,
	CONSTRAINT generator_pk PRIMARY KEY (generator_id)
);

ALTER TABLE powerplant.generators OWNER TO operator;
	
