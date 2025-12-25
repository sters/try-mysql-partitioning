-- Initialize database
CREATE DATABASE IF NOT EXISTS docmanager;
USE docmanager;

-- Create documents table with RANGE partitioning by year
CREATE TABLE IF NOT EXISTS documents (
    id INT AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id, created_at)
) ENGINE=InnoDB
PARTITION BY RANGE (YEAR(created_at)) (
    PARTITION p2020 VALUES LESS THAN (2021),
    PARTITION p2021 VALUES LESS THAN (2022),
    PARTITION p2022 VALUES LESS THAN (2023),
    PARTITION p2023 VALUES LESS THAN (2024),
    PARTITION p2024 VALUES LESS THAN (2025),
    PARTITION p2025 VALUES LESS THAN (2026),
    PARTITION pfuture VALUES LESS THAN MAXVALUE
);

-- Create attributes table with HASH partitioning
CREATE TABLE IF NOT EXISTS attributes (
    id INT AUTO_INCREMENT,
    document_id INT NOT NULL,
    attr_key VARCHAR(100) NOT NULL,
    attr_value TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, document_id),
    INDEX idx_document_id (document_id),
    INDEX idx_attr_key (attr_key)
) ENGINE=InnoDB
PARTITION BY HASH(document_id)
PARTITIONS 4;

-- Insert sample data
INSERT INTO documents (title, content, created_at) VALUES
    ('Document 2023', 'This is a document from 2023', '2023-01-15 10:00:00'),
    ('Document 2024', 'This is a document from 2024', '2024-06-20 14:30:00'),
    ('Document 2025', 'This is a document from 2025', '2025-12-01 09:15:00');

INSERT INTO attributes (document_id, attr_key, attr_value) VALUES
    (1, 'author', 'John Doe'),
    (1, 'category', 'Technical'),
    (2, 'author', 'Jane Smith'),
    (2, 'category', 'Business'),
    (3, 'author', 'Bob Johnson'),
    (3, 'category', 'General');
