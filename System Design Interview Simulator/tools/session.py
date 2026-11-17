#!/usr/bin/env python3
"""CLI-раннер сессии System Design интервью.

Только stdlib. Реализует гейт «эталон — только после попытки»:
`reveal` откажется показывать решение, если для сценария нет зафиксированной попытки
(обойти можно флагом --force, но это осознанное нарушение правил тренировки).

Команды:
    list                              — список сценариев
    start <slug>                      — показать бриф и вопросы (без решения)
    submit <slug> --file <path>       — сохранить попытку
    reveal <slug> [--force]           — показать эталон (после попытки)
    score <slug> --scores 3,4,2,3,3,3 — посчитать балл и грейд
    validate                          — валидация банка сценариев
"""
from __future__ import annotations

import argparse
import json
import sys
from datetime import date
from pathlib import Path

import validate as validate_mod

BASE = Path(__file__).resolve().parent.parent


def _load_json(path: Path) -> dict:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def load_bank() -> dict:
    return _load_json(BASE / "data" / "scenarios.json")


def load_card() -> dict:
    return _load_json(BASE / "rubric" / "scorecard.json")


def find(bank: dict, slug: str) -> dict | None:
    for sc in bank.get("scenarios", []):
        if sc.get("slug") == slug:
            return sc
    return None


def attempt_files(slug: str) -> list[Path]:
    d = BASE / "attempts"
    if not d.exists():
        return []
    return sorted(p for p in d.glob(f"*{slug}*") if p.name != ".gitkeep")


def cmd_list(bank: dict, _args) -> int:
    print(f"{'#':>2}  {'slug':<22} {'level':<12} title")
    print("-" * 70)
    for sc in bank["scenarios"]:
        print(f"{sc['order']:>2}  {sc['slug']:<22} {sc.get('level', ''):<12} {sc['title']}")
    print(f"\nВсего: {len(bank['scenarios'])}. Старт: session.py start <slug>")
    return 0


def cmd_start(bank: dict, args) -> int:
    sc = find(bank, args.slug)
    if not sc:
        print(f"Неизвестный сценарий: {args.slug}", file=sys.stderr)
        return 1
    card_path = BASE / sc["scenario_path"]
    print(card_path.read_text(encoding="utf-8"))
    print("\n--- Вопросы интервьюера (для самопроверки) ---")
    for i, q in enumerate(sc.get("interviewer_questions", []), 1):
        print(f"{i}. {q}")
    print(f"\nКогда зафиксируете ответ: session.py submit {sc['slug']} --file <ваш файл>")
    print(f"Затем: session.py reveal {sc['slug']}")
    return 0


def cmd_submit(bank: dict, args) -> int:
    sc = find(bank, args.slug)
    if not sc:
        print(f"Неизвестный сценарий: {args.slug}", file=sys.stderr)
        return 1
    src = Path(args.file)
    if not src.exists():
        print(f"Файл попытки не найден: {src}", file=sys.stderr)
        return 1
    out_dir = BASE / "attempts"
    out_dir.mkdir(exist_ok=True)
    dst = out_dir / f"{sc['order']:02d}-{sc['slug']}-{date.today().isoformat()}.md"
    dst.write_text(src.read_text(encoding="utf-8"), encoding="utf-8")
    print(f"Попытка сохранена: {dst.relative_to(BASE)}")
    print(f"Теперь можно открыть эталон: session.py reveal {sc['slug']}")
    return 0


def cmd_reveal(bank: dict, args) -> int:
    sc = find(bank, args.slug)
    if not sc:
        print(f"Неизвестный сценарий: {args.slug}", file=sys.stderr)
        return 1
    attempts = attempt_files(args.slug)
    if not attempts and not args.force:
        print(
            "ГЕЙТ: попытка не зафиксирована, эталон скрыт.\n"
            f"Сначала: session.py start {sc['slug']}  →  session.py submit {sc['slug']} --file <файл>\n"
            "Осознанный обход (не рекомендуется): reveal --force",
            file=sys.stderr,
        )
        return 2
    sol = (BASE / sc["solution_path"]).read_text(encoding="utf-8")
    print(sol)
    print(f"\n--- Red flags сценария ---")
    for rf in sc.get("red_flags", []):
        print(f"  ✗ {rf}")
    print(f"--- Strong signals ---")
    for ss in sc.get("strong_signals", []):
        print(f"  ✓ {ss}")
    return 0


def grade_for(total: float, grades: list[dict]) -> str:
    for g in sorted(grades, key=lambda x: -x["min_total"]):
        if total >= g["min_total"]:
            return g["grade"]
    return "not_ready"


def cmd_score(bank: dict, args) -> int:
    sc = find(bank, args.slug)
    if not sc:
        print(f"Неизвестный сценарий: {args.slug}", file=sys.stderr)
        return 1
    card = load_card()
    dims = card["dimensions"]
    try:
        scores = [int(x) for x in args.scores.split(",")]
    except ValueError:
        print("--scores должен быть списком целых через запятую, напр. 3,4,2,3,3,3", file=sys.stderr)
        return 1
    if len(scores) != len(dims):
        print(f"Ожидалось {len(dims)} оценок (по измерениям), получено {len(scores)}.", file=sys.stderr)
        print("Порядок измерений:")
        for i, d in enumerate(dims):
            print(f"  {i + 1}. {d['id']} ({d['title']}, вес {d['weight']})")
        return 1
    lo, hi = card["scale"]["min"], card["scale"]["max"]
    if any(s < lo or s > hi for s in scores):
        print(f"Оценки должны быть в диапазоне {lo}..{hi}", file=sys.stderr)
        return 1

    total = 0.0
    print(f"{'измерение':<28}{'оценка':>7}{'вес':>6}{'вклад':>8}")
    print("-" * 52)
    for d, s in zip(dims, scores):
        contribution = s / hi * d["weight"]
        total += contribution
        print(f"{d['id']:<28}{s:>5}/{hi}{d['weight']:>6}{contribution:>8.1f}")
    total = round(total, 1)
    grade = grade_for(total, card["grades"])
    print("-" * 52)
    print(f"{'ИТОГ':<28}{'':>7}{'':>6}{total:>8.1f}  ({grade})")

    record = {
        "scenario_slug": sc["slug"],
        "candidate": args.candidate,
        "date": date.today().isoformat(),
        "scores": {d["id"]: s for d, s in zip(dims, scores)},
        "total": total,
        "grade": grade,
        "notes": args.notes or "",
    }
    out_dir = BASE / "attempts"
    out_dir.mkdir(exist_ok=True)
    dst = out_dir / f"{sc['order']:02d}-{sc['slug']}-{date.today().isoformat()}.score.json"
    dst.write_text(json.dumps(record, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"Оценочный лист сохранён: {dst.relative_to(BASE)}")
    return 0


def cmd_validate(_bank: dict, _args) -> int:
    problems = validate_mod.validate_bank(BASE)
    if problems:
        print("НАЙДЕНЫ ПРОБЛЕМЫ:")
        for p in problems:
            print(f"  - {p}")
        return 1
    print("OK: банк валиден.")
    return 0


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="System Design Interview Simulator")
    sub = p.add_subparsers(dest="cmd", required=True)

    sub.add_parser("list", help="список сценариев").set_defaults(func=cmd_list)

    sp = sub.add_parser("start", help="начать попытку (без эталона)")
    sp.add_argument("slug")
    sp.set_defaults(func=cmd_start)

    sp = sub.add_parser("submit", help="сохранить попытку")
    sp.add_argument("slug")
    sp.add_argument("--file", required=True, help="файл с вашим ответом")
    sp.set_defaults(func=cmd_submit)

    sp = sub.add_parser("reveal", help="показать эталон (после попытки)")
    sp.add_argument("slug")
    sp.add_argument("--force", action="store_true", help="обойти гейт (не рекомендуется)")
    sp.set_defaults(func=cmd_reveal)

    sp = sub.add_parser("score", help="посчитать балл и грейд")
    sp.add_argument("slug")
    sp.add_argument("--scores", required=True, help="6 целых через запятую, напр. 3,4,2,3,3,3")
    sp.add_argument("--candidate", default="candidate")
    sp.add_argument("--notes", default="")
    sp.set_defaults(func=cmd_score)

    sub.add_parser("validate", help="валидация банка").set_defaults(func=cmd_validate)
    return p


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    if args.cmd == "validate":
        return cmd_validate({}, args)
    bank = load_bank()
    return args.func(bank, args)


if __name__ == "__main__":
    sys.exit(main())
