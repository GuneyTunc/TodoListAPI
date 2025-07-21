#!/bin/bash
# SQL Server'ı arka planda başlat
echo "SQL Server sürecini başlatıyorum..."
/opt/mssql/bin/sqlservr &
PID=$! # SQL Server sürecinin PID'sini al

echo "SQL Server'ın hazır olmasını bekliyorum (sqlcmd ile kontrol ediliyor)..."

# SQL Server'ın bağlantıları kabul etmesini bekle
for i in {1..60}; do
    /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "$SA_PASSWORD" -Q "SELECT 1" -l 30 -h -1 > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo "SQL Server hazır."
        break
    fi
    echo "SQL Server henüz hazır değil. 5 saniye daha bekliyorum... (Deneme $i/60)"
    sleep 5
done

# Son bir kontrol
/opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "$SA_PASSWORD" -Q "SELECT 1" > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "HATA: SQL Server'a bağlanılamadı. Logları kontrol edin."
    exit 1
fi

echo "init.sql dosyasını çalıştırıyorum (kullanıcı/login oluşturma kısmı hariç)."
# init.sql script'ini çalıştır (varsa sadece veritabanı/tablo oluşturma ve izinler)
# -v DB_PASSWORD parametresi artık init.sql içinde kullanılmayacağı için kaldırıldı.
/opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "$SA_PASSWORD" -i /usr/src/app/init/init.sql

echo "app_user login ve kullanıcısını oluşturuyorum/güncelliyorum."

# CREATE LOGIN komutunu oluştur ve çalıştır
# Şifreyi doğrudan komuta eklerken doğru şekilde tırnak içine alıyoruz.
CREATE_LOGIN_SQL="CREATE LOGIN app_user WITH PASSWORD = '$DB_PASSWORD', CHECK_POLICY = OFF;"
/opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "$SA_PASSWORD" -Q "$CREATE_LOGIN_SQL"

# CREATE USER komutunu oluştur ve çalıştır
# Önce veritabanı bağlamını değiştiriyoruz, sonra kullanıcıyı oluşturuyoruz.
CREATE_USER_SQL="USE TodoAppDB; CREATE USER app_user FOR LOGIN app_user;"
/opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "$SA_PASSWORD" -Q "$CREATE_USER_SQL"

echo "init.sql başarıyla çalıştırıldı. SQL Server'ı çalışır durumda tutuyorum."
wait $PID # SQL Server sürecini bekleyerek container'ı çalışır durumda tut