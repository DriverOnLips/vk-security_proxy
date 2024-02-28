CREATE TABLE IF NOT EXISTS requests (
    id SERIAL PRIMARY KEY,
    method VARCHAR(10),
    path VARCHAR(255),
    get_params JSONB,
    headers JSONB,
    cookies JSONB,
    post_params JSONB
);

CREATE TABLE IF NOT EXISTS responses (
    id SERIAL PRIMARY KEY,
    request_id INT,
    code INT,
    message VARCHAR(255),
    headers JSONB,
    body TEXT,
    FOREIGN KEY (request_id) REFERENCES requests(id)
);
