ALTER TABLE devices ADD CONSTRAINT uq_devices_user_model UNIQUE (user_id, model);
