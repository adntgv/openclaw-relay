const { createHash } = require('node:crypto');

const WS_GUID = '258EAFA5-E914-47DA-95CA-C5AB0DC85B11';

function createAcceptValue(secWebSocketKey) {
  return createHash('sha1')
    .update(`${secWebSocketKey}${WS_GUID}`)
    .digest('base64');
}

function encodeTextFrame(text) {
  const payload = Buffer.from(text, 'utf8');
  const length = payload.length;

  let header;
  if (length < 126) {
    header = Buffer.from([0x81, length]);
  } else if (length < 65536) {
    header = Buffer.alloc(4);
    header[0] = 0x81;
    header[1] = 126;
    header.writeUInt16BE(length, 2);
  } else {
    header = Buffer.alloc(10);
    header[0] = 0x81;
    header[1] = 127;
    header.writeBigUInt64BE(BigInt(length), 2);
  }

  return Buffer.concat([header, payload]);
}

class WebSocketFrameParser {
  constructor(onTextFrame) {
    this.buffer = Buffer.alloc(0);
    this.onTextFrame = onTextFrame;
  }

  push(chunk) {
    this.buffer = Buffer.concat([this.buffer, chunk]);

    while (this.buffer.length >= 2) {
      const byte1 = this.buffer[0];
      const byte2 = this.buffer[1];
      const opcode = byte1 & 0x0f;
      const isMasked = (byte2 & 0x80) !== 0;
      let payloadLength = byte2 & 0x7f;
      let offset = 2;

      if (payloadLength === 126) {
        if (this.buffer.length < 4) return;
        payloadLength = this.buffer.readUInt16BE(2);
        offset = 4;
      } else if (payloadLength === 127) {
        if (this.buffer.length < 10) return;
        const len64 = this.buffer.readBigUInt64BE(2);
        payloadLength = Number(len64);
        offset = 10;
      }

      const maskLength = isMasked ? 4 : 0;
      const frameLength = offset + maskLength + payloadLength;
      if (this.buffer.length < frameLength) return;

      let payloadStart = offset;
      let mask;
      if (isMasked) {
        mask = this.buffer.subarray(offset, offset + 4);
        payloadStart += 4;
      }

      const payload = this.buffer.subarray(payloadStart, payloadStart + payloadLength);

      if (opcode === 0x8) {
        this.buffer = this.buffer.subarray(frameLength);
        return;
      }

      if (opcode === 0x1) {
        const decoded = Buffer.from(payload);
        if (isMasked) {
          for (let i = 0; i < decoded.length; i += 1) {
            decoded[i] ^= mask[i % 4];
          }
        }
        this.onTextFrame(decoded.toString('utf8'));
      }

      this.buffer = this.buffer.subarray(frameLength);
    }
  }
}

function writeTextFrame(socket, text) {
  socket.write(encodeTextFrame(text));
}

module.exports = {
  createAcceptValue,
  WebSocketFrameParser,
  writeTextFrame,
};
