#!/bin/bash

# Test çalıştırma script'i
# Bu script test veritabanını hazırlar ve tüm testleri çalıştırır

set -e

echo "🧪 Bank App Test Suite"
echo "========================"

# Test veritabanı bağlantısını kontrol et
echo "📊 Test veritabanı bağlantısı kontrol ediliyor..."

# PostgreSQL servisinin çalışıp çalışmadığını kontrol et
if ! pg_isready -h localhost -p 5432 -U postgres > /dev/null 2>&1; then
    echo "❌ PostgreSQL servisi çalışmıyor!"
    echo "   PostgreSQL'i başlatın ve tekrar deneyin."
    exit 1
fi

# Test veritabanını oluştur
echo "🗄️  Test veritabanı hazırlanıyor..."
psql -h localhost -U postgres -d postgres -c "DROP DATABASE IF EXISTS bankapp_test;" > /dev/null 2>&1 || true
psql -h localhost -U postgres -d postgres -c "CREATE DATABASE bankapp_test;" > /dev/null 2>&1 || true

echo "✅ Test veritabanı hazırlandı"

# Go testlerini çalıştır
echo "🚀 Testler çalıştırılıyor..."

# Repository testleri
echo "📦 Repository testleri..."
go test -v ./test/infrastructure/... -timeout 30s

# Service testleri
echo "🔧 Service testleri..."
go test -v ./test/service/... -timeout 30s

# Coverage raporu
echo "📊 Coverage raporu oluşturuluyor..."
go test -v -coverprofile=coverage.out ./test/... -timeout 30s

# Coverage raporunu göster
if command -v go tool cover > /dev/null 2>&1; then
    echo "📈 Coverage özeti:"
    go tool cover -func=coverage.out
    echo ""
    echo "📋 Coverage detayları:"
    go tool cover -html=coverage.out -o coverage.html
    echo "📄 HTML raporu: coverage.html"
else
    echo "⚠️  go tool cover bulunamadı"
fi

echo ""
echo "✅ Tüm testler tamamlandı!"
echo "🎉 Test suite başarıyla çalıştırıldı." 