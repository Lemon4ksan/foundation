import os

fpath = r'd:\CodingProjects\foundation\net\quic\sys_conn.go'
with open(fpath, 'r', encoding='utf-8') as f:
    text = f.read()

# Replace slog.Warn("%s. See ...", err) with proper slog.Warn(...)
text = text.replace('slog.Warn(\n\t\t\t\t\t"%s. See https://github.com/lemon4ksan/sein/internal/quic/wiki/UDP-Buffer-Sizes for details.",\n\t\t\t\t\terr,\n\t\t\t\t)', 'slog.Warn("Failed to set buffer. See https://github.com/lemon4ksan/sein/internal/quic/wiki/UDP-Buffer-Sizes for details.", "error", err)')

text = text.replace('slog.Warn(\n\t\t\t\t\t"%s. See https://github.com/lemon4ksan/sein/internal/quic/wiki/UDP-Buffer-Sizes for details.",\n\t\t\t\t\terr,\n\t\t\t\t)', 'slog.Warn("Failed to set buffer. See https://github.com/lemon4ksan/sein/internal/quic/wiki/UDP-Buffer-Sizes for details.", "error", err)')

with open(fpath, 'w', encoding='utf-8') as f:
    f.write(text)

fpath = r'd:\CodingProjects\foundation\net\quic\sys_conn_oob.go'
with open(fpath, 'r', encoding='utf-8') as f:
    text = f.read()

text = text.replace('slog.Warn("Received invalid IPv4 packet info control message: %+x. "', 'slog.Warn("Received invalid IPv4 packet info control message", "data", ')
text = text.replace('slog.Warn("Received invalid IPv6 packet info control message: %+x. "', 'slog.Warn("Received invalid IPv6 packet info control message", "data", ')

with open(fpath, 'w', encoding='utf-8') as f:
    f.write(text)
