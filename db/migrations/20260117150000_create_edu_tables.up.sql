CREATE TABLE IF NOT EXISTS edu_assignments (
    id SERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    deadline_unix BIGINT,
    created_unix BIGINT,
    updated_unix BIGINT
);

CREATE TABLE IF NOT EXISTS edu_submissions (
    id SERIAL PRIMARY KEY,
    assignment_id BIGINT NOT NULL REFERENCES edu_assignments(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    student_repo_id BIGINT,
    status VARCHAR(50) NOT NULL DEFAULT 'started',
    created_unix BIGINT,
    updated_unix BIGINT
);

CREATE TABLE IF NOT EXISTS edu_test_results (
    id SERIAL PRIMARY KEY,
    submission_id BIGINT NOT NULL REFERENCES edu_submissions(id) ON DELETE CASCADE,
    commit_sha VARCHAR(64) NOT NULL,
    score INT DEFAULT 0,
    details TEXT,
    created_unix BIGINT
);

CREATE INDEX idx_edu_assignments_repo_id ON edu_assignments(repo_id);
CREATE INDEX idx_edu_submissions_assignment_id ON edu_submissions(assignment_id);
CREATE INDEX idx_edu_submissions_user_id ON edu_submissions(user_id);
