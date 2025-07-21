#!/bin/bash
# SQL Server'ı arka planda başlat
echo "SQL Server sürecini başlatıyorum..."
/opt/mssql/bin/sqlservr &
PID=$! # SQL Server sürecinin PID'sini al

echo "SQL Server'ın hazır olmasını bekliyorum (sqlcmd ile kontrol ediliyor)..."

# SQL Server'ın bağlantıları kabul etmesini bekle
# -l 30: 30 saniye bağlantı zaman aşımı
# -h -1: Başlıkları gizle
# -Q "SELECT 1": Basit bir sorgu çalıştır
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


echo "init.sql dosyasını çalıştırıyorum."
# init.sql script'ini çalıştır
/opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "$SA_PASSWORD" -i /usr/src/app/init/init.sql

echo "init.sql başarıyla çalıştırıldı. SQL Server'ı çalışır durumda tutuyorum."

# SQL Server sürecinin çalışmaya devam etmesini sağla
wait $PID