PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE host_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER,
			computer_name TEXT,
			fqdn TEXT,
			domain_name TEXT,
			os_name TEXT,
			os_edition TEXT,
			os_version TEXT,
			os_build TEXT,
			install_date_str TEXT,
			architecture TEXT,
			product_id TEXT,
			machine_guid TEXT,
			user_login TEXT,
			user_fullname TEXT,
			user_domain TEXT,
			user_join_type TEXT,
			bios_vendor TEXT,
			bios_version TEXT,
			bios_release_date TEXT,
			baseboard_vendor TEXT,
			baseboard_product TEXT,
			baseboard_version TEXT,
			baseboard_serial TEXT,
			cpu_name TEXT,
			cpu_cores INTEGER,
			cpu_threads INTEGER,
			cpu_speed_mhz INTEGER,
			cpu_l1_bytes INTEGER,
			cpu_l2_bytes INTEGER,
			cpu_l3_bytes INTEGER,
			ram_total_bytes INTEGER,
			ram_free_bytes INTEGER,
			client_ip TEXT
		);
INSERT INTO host_metrics VALUES(1,1785319117,'IT-S','IT-S.rsa.rogsibal.ru','rsa.rogsibal.ru','Windows 10 Pro','Professional','22H2','19045.6456','2024-11-05 18:30:20','amd64','00331-10000-00001-AA659','44CF1830-E856-4864-A4E7-E79DCE710F4B','Maxim.Alexandrov','Александров Максим Евгеньевич','rsa.rogsibal.ru','Domain','Hewlett-Packard','J61 v03.85','11/19/2014','Unknown Vendor','1589','0.00','Unknown/To be filled by O.E.M.','Intel(R) Xeon(R) CPU E5-1620 0 @ 3.60GHz',8,8,3591,262144,1048576,10485760,42869710848,17625247744,'172.17.113.11');
CREATE TABLE gpu_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host_id INTEGER,
			name TEXT,
			vendor_id INTEGER,
			device_id INTEGER,
			dedicated_memory INTEGER,
			shared_memory INTEGER,
			FOREIGN KEY(host_id) REFERENCES host_metrics(id) ON DELETE CASCADE
		);
INSERT INTO gpu_metrics VALUES(1,1,'NVIDIA NVS 510',4318,4093,2147483648,0);
INSERT INTO gpu_metrics VALUES(2,1,'Microsoft Remote Display Adapter',0,0,0,0);
INSERT INTO gpu_metrics VALUES(3,1,'Microsoft Remote Display Adapter',0,0,0,0);
INSERT INTO gpu_metrics VALUES(4,1,'Microsoft Remote Display Adapter',0,0,0,0);
CREATE TABLE disk_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host_id INTEGER,
			drive TEXT,
			type TEXT,
			vendor TEXT,
			serial_number TEXT,
			total_bytes INTEGER,
			free_bytes INTEGER,
			FOREIGN KEY(host_id) REFERENCES host_metrics(id) ON DELETE CASCADE
		);
INSERT INTO disk_metrics VALUES(1,1,'C:\','HDD','Unknown Vendor (Requires Admin)','Unknown S/N (Requires Admin)',498951073792,207509553152);
INSERT INTO disk_metrics VALUES(2,1,'D:\','CDROM','Unknown Vendor (Requires Admin)','Unknown S/N (Requires Admin)',250806272,0);
INSERT INTO disk_metrics VALUES(3,1,'G:\','HDD','Unknown Vendor (Requires Admin)','Unknown S/N (Requires Admin)',2000396742656,1861624500224);
CREATE TABLE network_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host_id INTEGER,
			name TEXT,
			mac TEXT,
			status TEXT,
			ip_type TEXT,
			ip_addresses TEXT,
			FOREIGN KEY(host_id) REFERENCES host_metrics(id) ON DELETE CASCADE
		);
INSERT INTO network_metrics VALUES(1,1,'Подключение по локальной сети* 7','','down','static','');
INSERT INTO network_metrics VALUES(2,1,'Подключение по локальной сети* 9','','down','dhcp','');
INSERT INTO network_metrics VALUES(3,1,'Беспроводная сеть','','down','dhcp','');
INSERT INTO network_metrics VALUES(4,1,'Подключение по локальной сети* 1','','down','static','');
INSERT INTO network_metrics VALUES(5,1,'vSwitch (WSL)','','down','static','');
INSERT INTO network_metrics VALUES(6,1,'Подключение по локальной сети* 5','','down','static','');
INSERT INTO network_metrics VALUES(7,1,'vSwitch (Default Switch)','','down','static','');
INSERT INTO network_metrics VALUES(8,1,'Подключение по локальной сети* 4','','down','static','');
INSERT INTO network_metrics VALUES(9,1,'Подключение по локальной сети* 8','','down','static','');
INSERT INTO network_metrics VALUES(10,1,'Подключение по локальной сети* 3','','down','static','');
INSERT INTO network_metrics VALUES(11,1,'vEthernet (Default Switch)','','up','static','172.30.224.1');
INSERT INTO network_metrics VALUES(12,1,'wsl-nic','','up','static','172.24.160.1');
INSERT INTO network_metrics VALUES(13,1,'Подключение по локальной сети* 6','','down','static','');
INSERT INTO network_metrics VALUES(14,1,'Сетевое подключение Bluetooth','','down','dhcp','');
INSERT INTO network_metrics VALUES(15,1,'Подключение по локальной сети* 2','','down','static','');
INSERT INTO network_metrics VALUES(16,1,'eth0','','up','static','172.17.113.11');
INSERT INTO network_metrics VALUES(17,1,'Ethernet (отладчик ядра)','','down','dhcp','');
INSERT INTO sqlite_sequence VALUES('host_metrics',1);
INSERT INTO sqlite_sequence VALUES('gpu_metrics',4);
INSERT INTO sqlite_sequence VALUES('disk_metrics',3);
INSERT INTO sqlite_sequence VALUES('network_metrics',17);
COMMIT;
