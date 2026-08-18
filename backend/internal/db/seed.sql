-- Datos de ejemplo. Los device_key coinciden con los que usa
-- cmd/simulator, así que si corres el simulador después de esto,
-- las posiciones empezarán a llegar automáticamente.

INSERT INTO tractors (name, plate, brand, model, device_key, status)
VALUES
    ('Tractor 01', 'ICA-101', 'John Deere', '6110M', 'GPS-0001', 'active'),
    ('Tractor 02', 'ICA-102', 'New Holland', 'T7.210', 'GPS-0002', 'active'),
    ('Tractor 03', 'ICA-103', 'Massey Ferguson', '5711', 'GPS-0003', 'active')
ON DUPLICATE KEY UPDATE name = VALUES(name);
