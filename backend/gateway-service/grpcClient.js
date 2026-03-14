const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const path = require('path');

const PROTO_PATH = path.join(__dirname, '../proto/scan.proto');
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});
const scanProto = grpc.loadPackageDefinition(packageDefinition).scan;

const client = new scanProto.ScanService(
  process.env.GRPC_HOST || 'threat-scan-service:50051',
  grpc.credentials.createInsecure()
);

module.exports = client;
