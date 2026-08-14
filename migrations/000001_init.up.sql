CREATE TABLE users (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    email         VARCHAR(255)    NOT NULL,
    password_hash VARCHAR(255)    NOT NULL,
    name          VARCHAR(100)    NOT NULL,
    created_at    TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_email (email)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE teams (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name       VARCHAR(100)    NOT NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_teams_created_by FOREIGN KEY (created_by) REFERENCES users (id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE team_members (
    team_id    BIGINT UNSIGNED                                           NOT NULL,
    user_id    BIGINT UNSIGNED                                           NOT NULL,
    role       ENUM ('owner', 'admin', 'member')                         NOT NULL DEFAULT 'member',
    created_at TIMESTAMP                                                 NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (team_id, user_id),
    CONSTRAINT fk_team_members_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE,
    CONSTRAINT fk_team_members_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE tasks (
    id          BIGINT UNSIGNED                             NOT NULL AUTO_INCREMENT,
    team_id     BIGINT UNSIGNED                             NOT NULL,
    title       VARCHAR(255)                                NOT NULL,
    description TEXT                                        NOT NULL,
    status      ENUM ('todo', 'in_progress', 'done')        NOT NULL DEFAULT 'todo',
    created_by  BIGINT UNSIGNED                             NOT NULL,
    assignee_id BIGINT UNSIGNED                             NULL,
    created_at  TIMESTAMP                                   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP                                   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    closed_at   TIMESTAMP                                   NULL,
    version     INT UNSIGNED                                NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    CONSTRAINT fk_tasks_team       FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE,
    CONSTRAINT fk_tasks_created_by FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT fk_tasks_assignee   FOREIGN KEY (assignee_id) REFERENCES users (id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_history (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    task_id    BIGINT UNSIGNED NOT NULL,
    changed_by BIGINT UNSIGNED NOT NULL,
    changes    JSON            NOT NULL,
    created_at TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_task_history_task FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_history_user FOREIGN KEY (changed_by) REFERENCES users (id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_comments (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    task_id    BIGINT UNSIGNED NOT NULL,
    user_id    BIGINT UNSIGNED NOT NULL,
    content    TEXT            NOT NULL,
    created_at TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_task_comments_task FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_comments_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE INDEX idx_teams_created_by ON teams (created_by);
CREATE INDEX idx_team_members_user ON team_members (user_id);
CREATE INDEX idx_tasks_team_status ON tasks (team_id, status);
CREATE INDEX idx_tasks_team_assignee ON tasks (team_id, assignee_id);
CREATE INDEX idx_tasks_team_created ON tasks (team_id, created_at);
CREATE INDEX idx_task_history_task ON task_history (task_id, created_at);
CREATE INDEX idx_task_comments_task ON task_comments (task_id, created_at);
