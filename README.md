# task-cli

Небольшой трекер задач на Go с интерфейсом командной строки. Задачи хранятся в
JSON-файле `tasks.json` в текущем рабочем каталоге, поэтому для работы не нужны
ни база данных, ни сервер.

Идея проекта: [roadmap.sh — Task Tracker](https://roadmap.sh/projects/task-tracker).

## Возможности

- Добавление задач с описанием и статусом.
- Список задач с фильтрацией по статусу.
- Изменение описания, смена статуса и удаление задач.
- Атомарная запись в `tasks.json` (через временный файл и переименование).
- Понятные сообщения об ошибках для неверных команд, статусов и id.

## Требования

- Go 1.27 или новее.

## Сборка

```bash
go build -o task-cli ./cmd/task-tracker
```

## Использование

Вывод команды `task-cli help`:

```bash
task-cli - manage simple tasks in tasks.json

Usage:
  task-cli add -d <description> [-s <status>]   add a task (status defaults to "todo")
  task-cli list [-s <status>]                   list tasks, optionally filtered by status
  task-cli update <id> -d <description>         change a task's description
  task-cli mark-done <id>                       set a task's status to "done"
  task-cli mark-in-progress <id>                set a task's status to "in progress"
  task-cli delete <id>                          remove a task
  task-cli help                                 show this message

Statuses: todo, in progress, done
```

Статус по умолчанию — `todo`. Парсер принимает его без учёта регистра и
понимает альтернативные написания: `in-progress`, `in_progress`, `doing` для
`in progress` и `complete` для `done`. Флаги можно писать как `-d "текст"`, так
и `-d="текст"`.

## Пример сессии

```bash
$ task-cli add -d "write the report"
added task 1
id:          1
description: write the report
status:      todo
created at:  2026-10-08 23:16:11
updated at:  2026-10-08 23:16:11

$ task-cli add -d "review pull request" -s "in progress"
added task 2
id:          2
description: review pull request
status:      in progress
created at:  2026-10-08 23:16:11
updated at:  2026-10-08 23:16:11

$ task-cli list
ID  STATUS       DESCRIPTION          CREATED              UPDATED
1   todo         write the report     2026-10-08 23:16:11  2026-10-08 23:16:11
2   in progress  review pull request  2026-10-08 23:16:11  2026-10-08 23:16:11

$ task-cli list -s done
no tasks found

$ task-cli mark-done 1
marked task 1 as done
id:          1
description: write the report
status:      done
created at:  2026-10-08 23:16:11
updated at:  2026-10-08 23:16:11

$ task-cli update 2 -d "review the pull request"
updated task 2
id:          2
description: review the pull request
status:      in progress
created at:  2026-10-08 23:16:11
updated at:  2026-10-08 23:16:11

$ task-cli delete 1
deleted task 1 (write the report)

$ task-cli list
ID  STATUS       DESCRIPTION              CREATED              UPDATED
2   in progress  review the pull request  2026-10-08 23:16:11  2026-10-08 23:16:11
```

## Хранение данных

Задачи сохраняются в `tasks.json` рядом с текущим каталогом:

```json
{
  "tasks": [
    {
      "id": 1,
      "description": "Test description",
      "status": "todo",
      "created_at": "2026-10-08T15:54:51Z",
      "updated_at": "2026-10-08T15:54:51Z"
    }
  ]
}
```

Если файла нет, он создаётся при первой записи. Идентификаторы не
переиспользуются: новый id всегда на единицу больше максимального.

## Структура проекта

```text
cmd/task-tracker/   точка входа и разбор аргументов командной строки
internal/task/      модель задачи, статусы и ошибки
internal/store/     загрузка, изменение и сохранение задач в JSON
```

## Тесты

```bash
go test ./...
```
