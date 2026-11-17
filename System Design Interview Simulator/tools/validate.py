#!/usr/bin/env python3
"""Структурная валидация банка сценариев и оценочного листа.

Только stdlib. Проверяет целостность data/scenarios.json:
наличие файлов scenario/solution, уникальность и порядок slug, обязательные поля,
непустые red_flags/strong_signals, корректность consistency.
Также проверяет, что scorecard.json содержит 6 измерений и веса в сумме 100.

Запуск:
    python3 tools/validate.py            # из каталога симулятора
    python3 tools/validate.py --root .   # явный корень
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

REQUIRED = [
    "slug", "order", "title", "level", "scenario_path", "solution_path",
    "functional", "non_functional", "interviewer_questions",
    "components", "consistency", "bottlenecks", "trade_offs",
    "red_flags", "strong_signals",
]


def _load_json(path: Path) -> dict:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def validate_bank(root: Path) -> list[str]:
    problems: list[str] = []
    bank_path = root / "data" / "scenarios.json"
    if not bank_path.exists():
        return [f"нет файла банка: {bank_path}"]

    bank = _load_json(bank_path)
    scenarios = bank.get("scenarios")
    if not isinstance(scenarios, list) or not scenarios:
        return ["в банке нет непустого списка scenarios"]

    orders: list[int] = []
    slugs: list[str] = []
    for i, sc in enumerate(scenarios):
        where = f"scenarios[{i}]"
        for key in REQUIRED:
            if key not in sc:
                problems.append(f"{where}: нет обязательного поля '{key}'")
        slug = sc.get("slug", f"<#{i}>")
        where = f"{where} ({slug})"

        orders.append(sc.get("order", -1))
        slugs.append(slug)

        for arr_key in ("functional", "non_functional", "interviewer_questions",
                        "components", "bottlenecks", "trade_offs", "red_flags", "strong_signals"):
            val = sc.get(arr_key)
            if not isinstance(val, list) or not val:
                problems.append(f"{where}: '{arr_key}' должен быть непустым списком")

        consistency = sc.get("consistency")
        if not isinstance(consistency, dict) or not consistency:
            problems.append(f"{where}: 'consistency' должен быть непустым словарём")
        elif any(not isinstance(v, str) or not v.strip() for v in consistency.values()):
            problems.append(f"{where}: значения 'consistency' должны быть непустыми строками")

        for pkey in ("scenario_path", "solution_path"):
            rel = sc.get(pkey)
            if isinstance(rel, str) and not (root / rel).exists():
                problems.append(f"{where}: файл '{rel}' ({pkey}) не найден")

        # slug должен совпадать с суффиксом имён файлов
        for pkey in ("scenario_path", "solution_path"):
            rel = sc.get(pkey)
            if isinstance(rel, str) and Path(rel).stem.split("-", 1)[-1] != slug:
                problems.append(f"{where}: slug не совпадает с именем файла '{rel}'")

    if len(set(slugs)) != len(slugs):
        problems.append("slug не уникальны")
    if orders != sorted(orders) or orders != list(range(1, len(scenarios) + 1)):
        problems.append(f"order должен быть 1..N по порядку, получено: {orders}")

    # scorecard
    card_path = root / "rubric" / "scorecard.json"
    if not card_path.exists():
        problems.append(f"нет файла: {card_path}")
    else:
        card = _load_json(card_path)
        dims = card.get("dimensions", [])
        if len(dims) != 6:
            problems.append(f"scorecard: ожидалось 6 измерений, найдено {len(dims)}")
        total_weight = sum(d.get("weight", 0) for d in dims)
        if total_weight != 100:
            problems.append(f"scorecard: сумма весов = {total_weight}, ожидалось 100")

    return problems


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Валидация банка сценариев симулятора")
    default_root = Path(__file__).resolve().parent.parent
    parser.add_argument("--root", type=Path, default=default_root, help="каталог симулятора")
    args = parser.parse_args(argv)

    problems = validate_bank(args.root)
    if problems:
        print("НАЙДЕНЫ ПРОБЛЕМЫ:")
        for p in problems:
            print(f"  - {p}")
        return 1
    print("OK: банк сценариев и оценочный лист валидны.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
