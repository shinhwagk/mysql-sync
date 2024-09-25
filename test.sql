




CREATE DATABASE IF NOT EXISTS test_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


USE test_db;


ALTER DATABASE test_db CHARACTER SET latin1 COLLATE latin1_swedish_ci;

CREATE TABLE users (
  id INT PRIMARY KEY,
  name VARCHAR(100),
  email VARCHAR(100) UNIQUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE orders (
  order_id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  amount DECIMAL(10,2),
  status ENUM('pending', 'completed', 'canceled') DEFAULT 'pending',
  order_date DATE,
  FOREIGN KEY (user_id) REFERENCES users(id)
);


CREATE TABLE users_backup LIKE users;

ALTER TABLE users ADD COLUMN age INT AFTER name;
ALTER TABLE users ADD COLUMN bio TEXT FIRST;

ALTER TABLE users MODIFY COLUMN email VARCHAR(150) NOT NULL;

ALTER TABLE users CHANGE COLUMN name full_name VARCHAR(200);

ALTER TABLE users DROP COLUMN bio;

ALTER TABLE users ADD INDEX idx_full_name (full_name);
ALTER TABLE users ADD UNIQUE INDEX uniq_email (email);

ALTER TABLE users DROP INDEX idx_full_name;

RENAME TABLE users TO app_users;

ALTER TABLE app_users CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

ALTER TABLE orders PARTITION BY RANGE (YEAR(order_date)) (
  PARTITION p0 VALUES LESS THAN (2020),
  PARTITION p1 VALUES LESS THAN (2021),
  PARTITION p2 VALUES LESS THAN MAXVALUE
);

CREATE INDEX idx_amount ON orders(amount);

DROP INDEX idx_amount ON orders;

CREATE USER 'new_user'@'localhost' IDENTIFIED BY 'password';

GRANT SELECT, INSERT ON test_db.* TO 'new_user'@'localhost';

REVOKE INSERT ON test_db.* FROM 'new_user'@'localhost';

ALTER USER 'new_user'@'localhost' IDENTIFIED BY 'new_password';

DROP USER 'new_user'@'localhost';

INSERT INTO app_users (id, full_name, email, age) VALUES (1, 'Alice', 'alice@example.com', 30);
INSERT INTO app_users (id, full_name, age) VALUES (2, 'Bob', 25);
INSERT INTO app_users SET id = 3, full_name = 'Charlie', email = 'charlie@example.com', age = 28;

UPDATE app_users SET email = 'alice_new@example.com' WHERE id = 1;
UPDATE app_users u JOIN orders o ON u.id = o.user_id SET u.age = u.age + 1 WHERE o.amount > 100;

DELETE FROM app_users WHERE id = 2;
DELETE u FROM app_users u JOIN orders o ON u.id = o.user_id WHERE o.status = 'canceled';

INSERT INTO app_users (id, full_name) VALUES (1, 'Alice Updated')
ON DUPLICATE KEY UPDATE full_name = VALUES(full_name);

REPLACE INTO app_users (id, full_name, email) VALUES (3, 'Charlie Replaced', 'charlie_new@example.com');

START TRANSACTION;
INSERT INTO app_users (id, full_name, age) VALUES (4, 'Dave', 22);
UPDATE app_users SET email = 'dave@example.com' WHERE id = 4;
COMMIT;

CREATE TABLE `select` (
  `from` VARCHAR(50),
  `to` VARCHAR(50)
);


CREATE TABLE `user-details` (
  `first-name` VARCHAR(50),
  `last-name` VARCHAR(50)
);


CREATE TABLE products (
  product_id INT PRIMARY KEY,
  name VARCHAR(100)
);

CREATE TABLE product_orders (
  order_id INT PRIMARY KEY,
  product_id INT,
  FOREIGN KEY (product_id) REFERENCES products(product_id)
);


ALTER TABLE product_orders
ADD CONSTRAINT fk_product
FOREIGN KEY (product_id) REFERENCES products(product_id);


ALTER TABLE product_orders DROP FOREIGN KEY fk_product;


CREATE TABLE test_generated (
  id INT PRIMARY KEY,
  name VARCHAR(100),
  name_length INT AS (CHAR_LENGTH(name)) STORED
);


CREATE TABLE test_json (
  id INT PRIMARY KEY,
  data JSON
);

INSERT INTO test_json (id, data) VALUES (1, '{ "key": "value" }');

ALTER TABLE app_users ADD CONSTRAINT chk_age CHECK (age >= 18);

ALTER TABLE orders REORGANIZE PARTITION p0 INTO (
  PARTITION p0a VALUES LESS THAN (2019),
  PARTITION p0b VALUES LESS THAN (2020)
);










TRUNCATE TABLE app_users;


 
 