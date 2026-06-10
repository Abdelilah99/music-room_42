CREATE UNIQUE INDEX uq_delegations_one_active ON delegations (device_id) WHERE revoked_at IS NULL;
