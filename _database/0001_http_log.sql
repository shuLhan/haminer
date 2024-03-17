DROP TABLE IF EXISTS http_log CASCADE;

CREATE TABLE http_log (
  request_date  TIMESTAMP WITH TIME ZONE

, client_ip     VARCHAR

, frontend_name VARCHAR
, backend_name  VARCHAR
, server_name   VARCHAR

, http_proto    VARCHAR
, http_method   VARCHAR
, http_url      VARCHAR
, http_query    VARCHAR

, header_request   VARCHAR
, header_response  VARCHAR

, cookie_request    VARCHAR
, cookie_response   VARCHAR
, termination_state VARCHAR

, bytes_read    BIGINT

, status_code   INTEGER
, client_port   INTEGER

, time_request  INTEGER
, time_wait     INTEGER
, time_connect  INTEGER
, time_response INTEGER
, time_all      INTEGER

, conn_active   INTEGER
, conn_frontend INTEGER
, conn_backend  INTEGER
, conn_server   INTEGER
, retries       INTEGER

, server_queue  INTEGER
, backend_queue INTEGER
);

DROP INDEX IF EXISTS http_log_idx;

CREATE INDEX IF NOT EXISTS http_log_idx ON http_log(
  request_date
, client_ip
, frontend_name
, backend_name
, server_name
, http_proto
, http_method
, http_url
, termination_state
, status_code
);

DROP INDEX IF EXISTS http_log_time_idx;

CREATE INDEX IF NOT EXISTS http_log_time_idx ON http_log(
  time_request
, time_wait
, time_connect
, time_response
, time_all
);
