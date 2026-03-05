<div>
  <h1 align="center">Stormlink ⚡</h1>
<p align="center"><i>Серверная часть децентрализованной платформы для создания сообществ.</i></p>
<div align="center">
<b>📢 Новости Stormic в telegram</b>
<br />
<a href='https://t.me/stormicapp'>https://t.me/stormicapp</a>
</div>
<br />
<b>Stormlink</b> — это серверная часть <b>Stormic Community Management System</b>. Платформы, которая объединяет людей для общения, создания контента и управления сообществами. Stormlink обеспечивает работу ядра экосистемы, связывая фронтенд (на NextJS) с функционалом мультиблогов, системы мгновенных сообщений и Wiki-модуля.
<br />
<br />
<div align="center">
<h2>📦 Микросервисы</h2>
<p><b>Основная разработка ведётся на GitLab:</b></p>
</div>
<br />
<table align="center">
  <tr>
    <td>🔐 <b>Auth Service</b></td>
    <td>Аутентификация и авторизация</td>
    <td><a href="https://gitlab.com/stormhead/stormlink/msa/auth">GitLab</a></td>
  </tr>
  <tr>
    <td>👤 <b>Profile Service</b></td>
    <td>Управление профилями пользователей</td>
    <td><a href="https://gitlab.com/stormhead/stormlink/msa/profile">GitLab</a></td>
  </tr>
  <tr>
    <td>📸 <b>Media Service</b></td>
    <td>Загрузка и обработка медиафайлов</td>
    <td><a href="https://gitlab.com/stormhead/stormlink/msa/media">GitLab</a></td>
  </tr>
  <tr>
    <td>👥 <b>Community Service</b></td>
    <td>Управление сообществами</td>
    <td><a href="https://gitlab.com/stormhead/stormlink/msa/community">GitLab</a></td>
  </tr>
  <tr>
    <td>🌐 <b>Instance Service</b></td>
    <td>Управление инстансами (сайтами)</td>
    <td><a href="https://gitlab.com/stormhead/stormlink/msa/instance">GitLab</a></td>
  </tr>
  <tr>
    <td>🔒 <b>Permissions Service</b></td>
    <td>Система разрешений</td>
    <td><a href="https://gitlab.com/stormhead/stormlink/msa/permissions">GitLab</a></td>
  </tr>
  <tr>
    <td>📦 <b>Protos</b></td>
    <td>Protocol Buffers определения</td>
    <td><a href="https://gitlab.com/stormhead/stormlink/msa/protos">GitLab</a></td>
  </tr>
</table>
<br />
<div align="center">
<h2>🛠 Технологии</h2>
</div>
<br />
<b>Stormlink</b> написан на <b>Go</b>.
<br />
- ✨ Использует <b>gRPC</b> для высокопроизводительного API и микросервисов.
<br />
- 🔄 Проксирует gRPC в <b>RESTful HTTP API</b> для взаимодействия с сервером сторонних сервисов.
<br />
- 📚 Соответствует спецификации <b>OpenAPI</b> и предоставляет <b>Swagger</b> документацию.
<br />
- 🗄️ Для работы с <b>PostgreSQL</b> использует <b>sqlc</b> — типобезопасная генерация SQL-кода.
<br />
- ✅ Использует <b>Protovalidate</b> для валидации входящих запросов на уровне proto.
<br />
- 🏗️ <b>Clean Architecture</b> — разделение на Entity, UseCase, Infrastructure, Controller.
<br />
- 📝 <b>slog</b> — структурированное логирование с контекстом.
<br />
- 🎫 <b>JWT</b> + <b>Refresh Token Rotation</b> для аутентификации.
<br />
- ☁️ <b>S3 / MinIO</b> для хранения медиафайлов.
<br />
<br />
<p align="center">
<strong>Stormic</strong> — мой пет-проект, который я делаю в свободное время.
</p>
<p align="center">
Контакты для связи:
<br/>
<a href='https://t.me/nimscore'>Telegram</a>
<br/>
<b>nimscore@gmail.com</b>
</p>
<p align="center">
<b>💰 ПОДДЕРЖАТЬ ПРОЕКТ:</b>
</p>
<table align="center">
  <tr>
    <td><b>₿ Bitcoin (BTC):</b></td>
    <td>bc1q0l6nenv6ts072y3dc98hmm4fcly4wwj4umnq95</td>
  </tr>
  <tr>
    <td><b>💎 TON (TON):</b></td>
    <td>UQBjnghaVNCvGmQRby8iBVNl9ifNCQV35YAKLxL6P0YuYrsh</td>
  </tr>
  <tr>
    <td><b>💠 ETH (Ethereum):</b></td>
    <td>0x79462b386494F9b039C544bFa5f77c2257bbE16b</td>
  </tr>
  <tr>
    <td><b>💵 USDT (Ethereum):</b></td>
    <td>0x79462b386494F9b039C544bFa5f77c2257bbE16b</td>
  </tr>
  <tr>
    <td><b>💵 USDT (TON):</b></td>
    <td>UQBjnghaVNCvGmQRby8iBVNl9ifNCQV35YAKLxL6P0YuYrsh</td>
  </tr>
  <tr>
    <td><b>🚀 Boosty:</b></td>
    <td><a href='https://boosty.to/nims'>https://boosty.to/nims</a></td>
  </tr>
</table>
<br/>
</div>
