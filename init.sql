-- Veritabanını oluştur (eğer yoksa)
IF NOT EXISTS (SELECT name FROM sys.databases WHERE name = 'TodoAppDB')
BEGIN
    CREATE DATABASE TodoAppDB;
    PRINT 'Database TodoAppDB created successfully.';
END
GO

USE TodoAppDB;
GO

-- TodoLists tablosu
IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='TodoLists' and xtype='U')
CREATE TABLE TodoLists (
    ID INT PRIMARY KEY IDENTITY(1,1),
    Title NVARCHAR(255) NOT NULL UNIQUE
);
GO

-- Todos tablosu
IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='Todos' and xtype='U')
CREATE TABLE Todos (
    ID INT PRIMARY KEY IDENTITY(1,1),
    ListID INT NOT NULL,
    Title NVARCHAR(255) NOT NULL,
    Description NVARCHAR(MAX),
    Completed BIT DEFAULT 0,
    CreatedAt DATETIME DEFAULT GETDATE(),
    UpdatedAt DATETIME DEFAULT GETDATE(),
    FOREIGN KEY (ListID) REFERENCES TodoLists(ID) ON DELETE CASCADE
);
GO

-- Todos tablosundaki ListID sütununa indeks ekle
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IX_Todos_ListID' AND object_id = OBJECT_ID('Todos'))
CREATE INDEX IX_Todos_ListID ON Todos (ListID);
GO

-- Todos tablosundaki Title sütununa indeks ekle (eğer sıkça başlığa göre arama yapılıyorsa)
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IX_Todos_Title' AND object_id = OBJECT_ID('Todos'))
CREATE INDEX IX_Todos_Title ON Todos (Title);
GO

-- Uygulama kullanıcısını ve login'ini tamamen silip yeniden oluştur
-- Bu, şifre senkronizasyonunu garanti eder
IF EXISTS (SELECT * FROM sys.database_principals WHERE name = 'app_user')
BEGIN
    DROP USER app_user;
    PRINT 'User app_user dropped.';
END
GO

IF EXISTS (SELECT * FROM sys.syslogins WHERE name = 'app_user')
BEGIN
    DROP LOGIN app_user;
    PRINT 'Login app_user dropped.';
END
GO

-- Uygulama kullanıcısı oluşturma
BEGIN
    -- DİKKAT: Şifre .env dosyasındaki DB_PASSWORD ile aynı olmalı!
    CREATE LOGIN app_user WITH PASSWORD = 'gGe2o05G!?', CHECK_POLICY = OFF;
    PRINT 'Login app_user created.';
END
GO

USE TodoAppDB;
GO

BEGIN
    CREATE USER app_user FOR LOGIN app_user;
    PRINT 'User app_user created in TodoAppDB.';
END
GO

-- Kullanıcıya gerekli izinleri ver
GRANT SELECT, INSERT, UPDATE, DELETE ON TodoLists TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON Todos TO app_user;
PRINT 'Permissions granted to app_user.';
GO

PRINT 'Veritabanı başlatma scripti tamamlandı.';