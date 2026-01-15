#!/bin/bash
set -e

echo "==> Generating SSL certificates for Kafka"

# Create secrets directory
mkdir -p secrets
cd secrets

# Configuration
VALIDITY_DAYS=365
KEYSTORE_PASSWORD="kafka-secret"
TRUSTSTORE_PASSWORD="kafka-secret"
KEY_PASSWORD="kafka-secret"
COUNTRY="US"
STATE="CA"
CITY="San Francisco"
ORG="Kafka Training"
OU="Engineering"
CN_CA="Kafka-CA"
CN_BROKER="kafka"

echo ""
echo "==> Step 1: Generate Certificate Authority (CA)"
# Generate CA private key
openssl req -new -x509 -keyout ca-key -out ca-cert -days $VALIDITY_DAYS \
  -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU/CN=$CN_CA" \
  -passout pass:$KEY_PASSWORD

echo ""
echo "==> Step 2: Generate Broker Keystore and Certificate"
# Create broker keystore with private key
keytool -genkey -noprompt \
  -alias kafka \
  -dname "C=$COUNTRY, ST=$STATE, L=$CITY, O=$ORG, OU=$OU, CN=$CN_BROKER" \
  -keystore kafka.keystore.jks \
  -keyalg RSA \
  -storepass $KEYSTORE_PASSWORD \
  -keypass $KEY_PASSWORD \
  -validity $VALIDITY_DAYS

# Create certificate signing request (CSR)
keytool -keystore kafka.keystore.jks -alias kafka -certreq -file cert-file \
  -storepass $KEYSTORE_PASSWORD -keypass $KEY_PASSWORD

# Sign the certificate with CA
openssl x509 -req -CA ca-cert -CAkey ca-key -in cert-file -out cert-signed \
  -days $VALIDITY_DAYS -CAcreateserial -passin pass:$KEY_PASSWORD

# Import CA certificate into keystore
keytool -keystore kafka.keystore.jks -alias CARoot -import -file ca-cert \
  -storepass $KEYSTORE_PASSWORD -noprompt

# Import signed certificate into keystore
keytool -keystore kafka.keystore.jks -alias kafka -import -file cert-signed \
  -storepass $KEYSTORE_PASSWORD -noprompt

echo ""
echo "==> Step 3: Generate Broker Truststore"
# Create truststore and import CA certificate
keytool -keystore kafka.truststore.jks -alias CARoot -import -file ca-cert \
  -storepass $TRUSTSTORE_PASSWORD -noprompt

echo ""
echo "==> Step 4: Generate Client Keystore (for mTLS)"
# Create client keystore with private key
keytool -genkey -noprompt \
  -alias client \
  -dname "C=$COUNTRY, ST=$STATE, L=$CITY, O=$ORG, OU=$OU, CN=kafka-client" \
  -keystore client.keystore.jks \
  -keyalg RSA \
  -storepass $KEYSTORE_PASSWORD \
  -keypass $KEY_PASSWORD \
  -validity $VALIDITY_DAYS

# Create client certificate signing request
keytool -keystore client.keystore.jks -alias client -certreq -file client-cert-file \
  -storepass $KEYSTORE_PASSWORD -keypass $KEY_PASSWORD

# Sign the client certificate with CA
openssl x509 -req -CA ca-cert -CAkey ca-key -in client-cert-file -out client-cert-signed \
  -days $VALIDITY_DAYS -CAcreateserial -passin pass:$KEY_PASSWORD

# Import CA certificate into client keystore
keytool -keystore client.keystore.jks -alias CARoot -import -file ca-cert \
  -storepass $KEYSTORE_PASSWORD -noprompt

# Import signed client certificate into keystore
keytool -keystore client.keystore.jks -alias client -import -file client-cert-signed \
  -storepass $KEYSTORE_PASSWORD -noprompt

echo ""
echo "==> Step 5: Generate Client Truststore"
# Create client truststore and import CA certificate
keytool -keystore client.truststore.jks -alias CARoot -import -file ca-cert \
  -storepass $TRUSTSTORE_PASSWORD -noprompt

echo ""
echo "==> Step 6: Export CA certificate in PEM format (for Go clients)"
# Already in PEM format from OpenSSL
cp ca-cert ca-cert.pem

echo ""
echo "==> Step 7: Create credential files for Confluent Kafka"
# Create password files for Confluent's Docker image
echo "kafka-secret" > keystore_creds
echo "kafka-secret" > key_creds
echo "kafka-secret" > truststore_creds

echo ""
echo "==> Cleanup temporary files"
rm -f cert-file cert-signed client-cert-file client-cert-signed ca-cert.srl

echo ""
echo "==> Certificate generation complete!"
echo ""
echo "Generated files in secrets/:"
echo "  - ca-cert, ca-key: Certificate Authority"
echo "  - ca-cert.pem: CA certificate in PEM format (for Go clients)"
echo "  - kafka.keystore.jks: Broker keystore"
echo "  - kafka.truststore.jks: Broker truststore"
echo "  - client.keystore.jks: Client keystore (for mTLS)"
echo "  - client.truststore.jks: Client truststore"
echo "  - keystore_creds, key_creds, truststore_creds: Password files for Confluent Kafka"
echo ""
echo "Passwords for all keystores and truststores: $KEYSTORE_PASSWORD"
echo ""
echo "To verify the broker certificate:"
echo "  keytool -list -v -keystore secrets/kafka.keystore.jks -storepass $KEYSTORE_PASSWORD"
echo ""
