# Менеджер ПВЗ (Пункт Выдачи Заказов)

Консольное приложение для управления пунктом выдачи заказов. Позволяет принимать заказы от курьеров, выдавать их клиентам и обрабатывать возвраты.

## Возможности

- Прием заказов от курьера (поштучно или из JSON-файла)
- Возврат заказов курьеру
- Выдача заказов клиентам
- Прием возвратов от клиентов
- Просмотр списка заказов с фильтрацией
- Просмотр списка возвратов с пагинацией
- Просмотр истории заказов
- Хранение данных в JSON-файле

## Команды приложения

1. **accept_order** - Принять заказ от курьера

```
accept_order <orderID> <clientID> <deadline> <weight> <cost> [package_type[+wrapper]]
```

- deadline: в формате "YYYY-MM-DDTHH:MM:SS" или как длительность (например, "48h")
- package_type: box/bag/film
- wrapper: +film (опционально)

2. **return_to_courier** - Вернуть заказ курьеру

```
return_to_courier <orderID>
```

3. **process_customer** - Выдать заказы или принять возврат

```
process_customer <customerID> <action> <orderID1> [orderID2 ...]
```

- action: handout/return

4. **list_orders** - Получить список заказов

```
list_orders <customerID> [pageSize <N>] [last <N>] [pvz]
```

5. **list_returns** - Получить список возвратов

```
list_returns pageSize <size>
```

6. **order_history** - Получить историю заказов

```
order_history
```

7. **accept_orders_file** - Принять заказы из JSON файла

```
accept_orders_file <filename>
```

8. Дополнительные команды:

- `help` - показать справку
- `clear` - очистить экран
- `exit` - выйти из программы
- `clear_db` - очистить базу данных

## Makefile команды

- `make build` - сборка проекта
- `make run` - сборка и запуск
- `make clean` - очистка артефактов сборки
- `make lint` - проверка кода линтерами
- `make fmt` - форматирование кода

## Формат JSON файла для импорта заказов

```json
[
  {
    "id": 1,
    "customer_id": 1,
    "deadline_at": "2030-02-20T15:04:05",
    "weight": 5.0,
    "cost": 100.0,
    "package_type": "box",
    "wrapper": "film"
  }
]
```
