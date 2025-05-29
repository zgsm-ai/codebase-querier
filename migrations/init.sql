-- goctl model pg datasource --dir internal/model  --style go_zero --url   postgres://root:password@127.0.0.1:5432/codebase_indexer?sslmode=disabl --table  --table codebase,sync_history,index_history
CREATE
database codebase_indexer with OWNER shenma;
-- Project repository table
CREATE TABLE codebase_indexer.public.codebase
(
    id             integer      NOT NULL,
    client_id      VARCHAR(255) NOT NULL, -- User client identifier, e.g., MAC address
    user_id        VARCHAR(255) NOT NULL, -- User identifier, e.g., email or phone number
    name           VARCHAR(255) NOT NULL, -- Codebase name
    local_path     TEXT         NOT NULL, -- Local path of the project
    path           TEXT         NOT NULL, -- Codebase path
    file_count     INT          NOT NULL,
    total_size     BIGINT       NOT NULL,
    extra_metadata JSON,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT
ON TABLE codebase_indexer.public.codebase IS 'Stores basic information about project repositories';
COMMENT
ON COLUMN codebase_indexer.public.codebase.id IS 'Unique identifier for the project repository';
COMMENT
ON COLUMN codebase_indexer.public.codebase.client_id IS 'User client identifier, such as MAC address';
COMMENT
ON COLUMN codebase_indexer.public.codebase.user_id IS 'User identifier, such as email or phone number';
COMMENT
ON COLUMN codebase_indexer.public.codebase.name IS 'Name of the project repository';
COMMENT
ON COLUMN codebase_indexer.public.codebase.local_path IS 'Local path of the project on the user''s machine';
COMMENT
ON COLUMN codebase_indexer.public.codebase.path IS 'Path of the codebase';
COMMENT
ON COLUMN codebase_indexer.public.codebase.file_count IS 'Number of files in the project';
COMMENT
ON COLUMN codebase_indexer.public.codebase.total_size IS 'Total size of the project (in bytes)';
COMMENT
ON COLUMN codebase_indexer.public.codebase.extra_metadata IS 'Additional metadata about the project';
COMMENT
ON COLUMN codebase_indexer.public.codebase.created_at IS 'Time when the record was created';
COMMENT
ON COLUMN codebase_indexer.public.codebase.updated_at IS 'Time when the record was last updated';

CREATE SEQUENCE codebase_indexer.public.codebase_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE CACHE 1;

ALTER SEQUENCE codebase_indexer.public.codebase_id_seq OWNED BY codebase_indexer.public.codebase.id;
ALTER TABLE ONLY codebase_indexer.public.codebase ALTER COLUMN id SET DEFAULT nextval('codebase_indexer.public.codebase_id_seq'::regclass);
ALTER TABLE ONLY codebase_indexer.public.codebase ADD CONSTRAINT codebase_pkey PRIMARY KEY (id);

-- Synchronization history table
CREATE TABLE codebase_indexer.public.sync_history
(
    id             integer     NOT NULL,
    codebase_id    INT         NOT NULL,                   -- codebase.id
    message        JSON        NOT NULL,
    publish_status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, success, failed
    publish_time   TIMESTAMP,
    created_at     TIMESTAMP            DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP            DEFAULT CURRENT_TIMESTAMP
);

COMMENT
ON TABLE codebase_indexer.public.sync_history IS 'Records the synchronization history of projects';
COMMENT
ON COLUMN codebase_indexer.public.sync_history.id IS 'Unique identifier for the synchronization history record';
COMMENT
ON COLUMN codebase_indexer.public.sync_history.codebase_id IS 'ID of the associated project repository';
COMMENT
ON COLUMN codebase_indexer.public.sync_history.message IS 'Content of the synchronization message';
COMMENT
ON COLUMN codebase_indexer.public.sync_history.publish_status IS 'Publishing status: pending, success, failed';
COMMENT
ON COLUMN codebase_indexer.public.sync_history.publish_time IS 'Time of publication';
COMMENT
ON COLUMN codebase_indexer.public.sync_history.created_at IS 'Time when the record was created';
COMMENT
ON COLUMN codebase_indexer.public.sync_history.updated_at IS 'Time when the record was last updated';

CREATE SEQUENCE codebase_indexer.public.sync_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE CACHE 1;

ALTER SEQUENCE codebase_indexer.public.sync_history_id_seq OWNED BY codebase_indexer.public.sync_history.id;
ALTER TABLE ONLY codebase_indexer.public.sync_history ALTER COLUMN id SET DEFAULT nextval('codebase_indexer.public.sync_history_id_seq'::regclass);
ALTER TABLE ONLY codebase_indexer.public.sync_history ADD CONSTRAINT sync_history_pkey PRIMARY KEY (id);

CREATE INDEX idx_sync_history_codebase_id ON codebase_indexer.public.sync_history USING btree (codebase_id);
CREATE INDEX idx_sync_history_publish_status ON codebase_indexer.public.sync_history USING btree (publish_status);

-- Index building task history table
CREATE TABLE codebase_indexer.public.index_history
(
    id                  integer     NOT NULL,
    sync_id             INTEGER     NOT NULL, -- sync_history.id
    codebase_id         INTEGER     not null, -- codebase.id
    codebase_path       TEXT        not null, -- codebase
    codebase_name       TEXT        not null, -- codebase
    total_file_count    INTEGER     NOT NULL,
    total_success_count INTEGER     NOT NULL,
    total_fail_count    INTEGER     NOT NULL,
    total_ignore_count  INTEGER     NOT NULL,
    task_type           VARCHAR(50) NOT NULL, -- vector, relation
    status              VARCHAR(50) NOT NULL, -- pending, running, success, failed
    progress            float,                -- index job progress
    error_message       TEXT,                 -- failed  message
    start_time          TIMESTAMP,
    end_time            TIMESTAMP,
    created_at          TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT
ON TABLE codebase_indexer.public.index_history IS 'Records the history of index building tasks';
COMMENT
ON COLUMN codebase_indexer.public.index_history.id IS 'Unique identifier for the index task history record';
COMMENT
ON COLUMN codebase_indexer.public.index_history.sync_id IS 'ID of the associated synchronization history record';
COMMENT
ON COLUMN codebase_indexer.public.index_history.codebase_id IS 'ID of the associated project repository';
COMMENT
ON COLUMN codebase_indexer.public.index_history.codebase_path IS 'Path of the project repository';
CoMMENT
ON COLUMN codebase_indexer.public.index_history.codebase_name IS 'name of the project repository';
COMMENT
ON COLUMN codebase_indexer.public.index_history.total_file_count IS 'Total number of files';
COMMENT
ON COLUMN codebase_indexer.public.index_history.total_success_count IS 'Total success number of files';
COMMENT
ON COLUMN codebase_indexer.public.index_history.total_fail_count IS 'Total fail number of files';
COMMENT
ON COLUMN codebase_indexer.public.index_history.total_ignore_count IS 'Total ignore number of files';
COMMENT
ON COLUMN codebase_indexer.public.index_history.task_type IS 'Task type: vector, relation';
COMMENT
ON COLUMN codebase_indexer.public.index_history.status IS 'Task status: pending, running, success, failed';
COMMENT
ON COLUMN codebase_indexer.public.index_history.progress IS 'Task progress (floating point number between 0 and 1)';
COMMENT
ON COLUMN codebase_indexer.public.index_history.error_message IS 'Error message if the task failed';
COMMENT
ON COLUMN codebase_indexer.public.index_history.start_time IS 'Task start time';
COMMENT
ON COLUMN codebase_indexer.public.index_history.end_time IS 'Task end time';
COMMENT
ON COLUMN codebase_indexer.public.index_history.created_at IS 'Time when the record was created';
COMMENT
ON COLUMN codebase_indexer.public.index_history.updated_at IS 'Time when the record was last updated';

CREATE SEQUENCE codebase_indexer.public.index_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE CACHE 1;

ALTER SEQUENCE codebase_indexer.public.index_history_id_seq OWNED BY codebase_indexer.public.index_history.id;
ALTER TABLE ONLY codebase_indexer.public.index_history ALTER COLUMN id SET DEFAULT nextval('codebase_indexer.public.index_history_id_seq'::regclass);
ALTER TABLE ONLY codebase_indexer.public.index_history ADD CONSTRAINT index_history_pkey PRIMARY KEY (id);