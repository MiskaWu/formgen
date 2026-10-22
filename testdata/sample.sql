CREATE TABLE demo_category (
    id INT PRIMARY KEY NOT NULL,
    icon VARCHAR(255),
    name VARCHAR(255),
    enabled BOOLEAN
);

CREATE TABLE demo_mission (
    id INT PRIMARY KEY NOT NULL,
    region_id INT,
    name_text INT,
    cost TINYINT,
    order_field INT,
    drops INT,
    suggest_power INT,
    scene_actor VARCHAR(255)
);

CREATE TABLE demo_rank_reward (
    id INT PRIMARY KEY NOT NULL,
    group_id INT,
    rank_max INT,
    rank_min INT,
    desc_text INT,
    mail_template INT,
    loot JSON COMMENT 'nested array field (stored as JSON)'
);

CREATE TABLE demo_region (
    id INT PRIMARY KEY NOT NULL,
    difficulty INT,
    order_field INT,
    name_text INT,
    desc_text INT,
    map_asset VARCHAR(255),
    tags JSON COMMENT 'array field (stored as JSON)',
    neighbors JSON COMMENT 'array field (stored as JSON)'
);

CREATE TABLE demo_shop (
    id INT PRIMARY KEY NOT NULL,
    label INT,
    sort_order INT,
    goods_group_id JSON COMMENT 'array field (stored as JSON)',
    display_type SMALLINT,
    pick_count SMALLINT,
    name_text INT
);

CREATE TABLE demo_shop_goods (
    id INT PRIMARY KEY NOT NULL,
    name_text INT,
    group_id SMALLINT,
    reward_type INT,
    reward_id INT,
    weight_field SMALLINT,
    amount INT,
    price_item INT,
    price INT,
    discount FLOAT,
    icon VARCHAR(255),
    purchase_limit INT
);

CREATE TABLE demo_shop_label (
    id INT PRIMARY KEY NOT NULL,
    label_text INT,
    label_order INT
);
