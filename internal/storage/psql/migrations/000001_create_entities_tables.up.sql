CREATE TABLE IF NOT EXISTS posts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL,
    body text NOT NULL,
    comments_enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

CREATE TABLE IF NOT EXISTS comments (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL,
    post_id uuid  NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    parent_id uuid REFERENCES comments(id) ON DELETE CASCADE,
    body varchar(2000) NOT NULL,
    created_at timestamptz  NOT NULL DEFAULT now(),

CONSTRAINT comments_post_id_id_unique UNIQUE (post_id, id),

CONSTRAINT comments_parent_same_post_fk
    FOREIGN KEY (post_id, parent_id)
    REFERENCES comments (post_id, id)
    ON DELETE CASCADE
);

CREATE INDEX comments_root_page_idx
        ON comments (post_id, id ASC)
    WHERE parent_id IS NULL;

CREATE INDEX comments_post_id_id_idx ON comments (post_id, id);

CREATE INDEX idx_comments_parent ON comments (parent_id);
CREATE INDEX idx_comments_post_parent ON comments (post_id, parent_id)
    WHERE parent_id IS NULL;


