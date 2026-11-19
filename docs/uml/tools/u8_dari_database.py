"""Membangkitkan docs/uml/U8-erd.puml dari skema database sungguhan.

ERD fisik (U8) harus sama dengan tabel yang benar-benar dibuat migrasi
(R-UML-03, D-80). Daripada menulisnya dengan tangan, skrip ini membaca
information_schema PostgreSQL lewat psql di container, lalu menulis satu
diagram PlantUML per kelompok tabel.

Cara pakai (dari root repo, PostgreSQL menyala, database sudah dimigrasi):

    python docs/uml/tools/u8_dari_database.py revioten

Argumen adalah nama database; bawaannya revioten. Database E2E
(revioten_e2e) juga boleh, karena skemanya sama.

Hanya membaca. Skrip ini tidak mengubah database.
"""

import subprocess
import sys
from collections import defaultdict
from datetime import date

DATABASE = sys.argv[1] if len(sys.argv) > 1 else "revioten"
OUTPUT = "docs/uml/U8-erd.puml"

# Kelompok diagram: judul dan tabelnya. Tabel yang belum masuk kelompok mana
# pun membuat skrip berhenti, supaya tabel baru tidak terlewat.
GROUPS = [
    ("U8-A-master-data", "Master data customer, model, dan proses", [
        "project_groups", "customers", "customer_codes", "model_groups", "models", "part_numbers",
        "processes", "process_names", "subprocesses"]),
    ("U8-B-akses-dan-audit", "Orang, akun, peran, hak, sesi, dan audit log", [
        "people", "accounts", "sessions", "roles", "permissions", "role_permissions", "account_roles", "audit_log",
        "system_settings"]),
    ("U8-C-perubahan-dan-rute", "Jenis dokumen, master perubahan, jabatan, dan rute approval", [
        "document_types", "special_codes", "change_types", "change_sources", "four_m_categories",
        "positions", "position_holders", "approval_routes", "route_slots", "route_slot_positions"]),
    ("U8-D-register-nomor", "Register nomor dokumen", [
        "number_settings", "number_counters", "number_batches", "document_numbers"]),
    ("U8-E-format-dan-form-set", "Format template, form set, dan persetujuan form", [
        "format_templates", "format_template_versions", "format_fields", "format_version_fields",
        "format_rating_criteria", "format_symbols", "form_sets", "form_set_versions", "form_approvals"]),
    ("U8-F-process-template", "Process template, poin, varian, dan dokumen terkait", [
        "process_templates", "process_template_versions", "process_template_subprocesses", "template_points",
        "process_template_points", "process_template_variants", "referenced_documents", "referenced_document_links"]),
    ("U8-G-dokumen", "Dokumen, baris, bagian per PCB, dan cek PFMEA-CP", [
        "documents", "document_models", "document_part_numbers", "document_rows", "document_sections",
        "document_section_part_numbers", "document_point_exclusions", "document_template_modes", "pfmea_cp_checks"]),
    ("U8-H-perubahan-revisi-yokoten", "Perubahan, revisi, rilis, yokoten, dan tanda tangan digital", [
        "changes", "change_documents", "change_attachments", "change_impacts", "document_revisions",
        "yokotens", "yokoten_targets", "yokoten_target_documents",
        "signature_requests", "signature_slots", "signatures"]),
    ("U8-I-cetak-dan-migrasi", "Cetak, layout, dan migrasi dokumen lama", [
        "print_layouts", "customer_print_layouts", "print_logs", "document_baselines", "document_history_entries"]),
    ("U8-J-integrasi-dan-notifikasi", "API integrasi, email notifikasi, dan dasbor admin", [
        "service_accounts", "service_account_customers", "service_tokens", "integration_requests",
        "change_proposals", "export_jobs", "notification_events", "notification_outbox",
        "periodic_reviews", "periodic_review_notices", "error_reports", "problem_reports"]),
]

SQL = """
WITH cols AS (
  SELECT c.table_name, c.column_name, c.ordinal_position, c.is_nullable,
         CASE WHEN c.data_type = 'ARRAY' THEN replace(c.udt_name, '_', '') || '[]'
              WHEN c.data_type = 'USER-DEFINED' THEN c.udt_name
              WHEN c.data_type = 'timestamp with time zone' THEN 'timestamptz'
              WHEN c.data_type = 'character varying' THEN 'varchar'
              ELSE c.data_type END AS type
  FROM information_schema.columns c
  WHERE c.table_schema = 'public' AND c.table_name <> 'goose_db_version'
),
keys AS (
  SELECT tc.table_name, kcu.column_name, tc.constraint_type,
         (SELECT count(*) FROM information_schema.key_column_usage k2
          WHERE k2.constraint_name = tc.constraint_name AND k2.table_schema = 'public') AS width
  FROM information_schema.table_constraints tc
  JOIN information_schema.key_column_usage kcu ON kcu.constraint_name = tc.constraint_name AND kcu.table_schema = tc.table_schema
  WHERE tc.table_schema = 'public' AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE', 'FOREIGN KEY')
)
SELECT 'C', c.table_name, c.column_name, c.type, c.is_nullable,
       coalesce(string_agg(DISTINCT CASE WHEN k.constraint_type = 'PRIMARY KEY' THEN 'PK'
                                         WHEN k.constraint_type = 'FOREIGN KEY' THEN 'FK'
                                         WHEN k.constraint_type = 'UNIQUE' AND k.width = 1 THEN 'UK' END, ','), ''),
       c.ordinal_position::text
FROM cols c LEFT JOIN keys k ON k.table_name = c.table_name AND k.column_name = c.column_name
GROUP BY c.table_name, c.column_name, c.type, c.is_nullable, c.ordinal_position
UNION ALL
SELECT DISTINCT 'F', tc.table_name, ccu.table_name, '', '', '', ''
FROM information_schema.table_constraints tc
JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = tc.constraint_name AND ccu.table_schema = tc.table_schema
WHERE tc.table_schema = 'public' AND tc.constraint_type = 'FOREIGN KEY'
UNION ALL
SELECT 'R', c.relname, '', '', '', '', ''
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relrowsecurity
UNION ALL
SELECT 'M', max(version_id)::text, '', '', '', '', '' FROM goose_db_version;
"""


def query():
    out = subprocess.run(
        ["docker", "compose", "exec", "-T", "db", "psql", "-U", "revioten", "-d", DATABASE, "-At", "-F", "\t", "-c", SQL],
        capture_output=True, text=True, encoding="utf-8", check=True)
    return [line.split("\t") for line in out.stdout.splitlines() if line]


def main():
    rows = query()
    columns = defaultdict(list)
    links = set()
    rls = set()
    migration = "?"
    for r in rows:
        kind = r[0]
        if kind == "C":
            columns[r[1]].append((int(r[6]), r[2], r[3], r[4] == "NO", r[5]))
        elif kind == "F" and r[1] != r[2]:
            links.add((r[1], r[2]))
        elif kind == "R":
            rls.add(r[1])
        elif kind == "M":
            migration = r[1]

    grouped = {t for _, _, tables in GROUPS for t in tables}
    missing = sorted(set(columns) - grouped)
    if missing:
        sys.exit("Tabel belum masuk kelompok mana pun di GROUPS: " + ", ".join(missing))

    lines = [
        f"' U8 · ERD fisik (draft 0.16, {date.today().isoformat()}), dibangkitkan dari skema database sampai migrasi {int(migration):05d}",
        "' oleh docs/uml/tools/u8_dari_database.py. Jangan diedit tangan: ubah migrasinya, lalu jalankan skrip itu lagi.",
        "' Menggambarkan tabel PostgreSQL 18 yang sudah dibangun (R-UML-03: diagram fisik mengikuti kode).",
        "' Tim IT boleh memakai stack lain (R-UML-02); aturan yang wajib dipertahankan ada di brief dan U5.",
        "'",
        "' Konvensi:",
        "'   * di depan kolom   = NOT NULL",
        "'   <<PK>> <<FK>> <<UK>> = primary key, foreign key, unique satu kolom",
        "'   <<RLS>> pada tabel = row-level security menyaring baris per akun (M8.1)",
        "'   garis menuju tabel induk yang dirujuk foreign key; tabel di kelompok lain ditampilkan ringkas",
        "'",
        "' Render dari folder docs/uml:",
        "'   java -jar plantuml-1.2026.9.jar -tsvg -o export U8-erd.puml",
        "",
    ]
    for diagram, title, tables in GROUPS:
        lines += [
            "' " + "-" * 75,
            f"@startuml {diagram}",
            "hide circle",
            "hide empty members",
            "skinparam linetype ortho",
            "skinparam shadowing false",
            "skinparam ArrowColor #444444",
            "skinparam ClassBorderColor #1F4E79",
            "skinparam ClassBackgroundColor #FFFFFF",
            "",
            f"title {diagram.split('-')[0]}-{diagram.split('-')[1]} · {title}",
            "",
        ]
        for t in tables:
            stereo = " <<RLS>>" if t in rls else ""
            lines.append(f'entity "{t}" as {t}{stereo} {{')
            cols = sorted(columns[t])
            pk = [c for c in cols if "PK" in c[4].split(",")]
            rest = [c for c in cols if "PK" not in c[4].split(",")]
            for _, name, typ, notnull, keys in pk:
                tags = " ".join(f"<<{k}>>" for k in keys.split(",") if k)
                lines.append(f"  * {name} : {typ} {tags}".rstrip())
            lines.append("  --")
            for _, name, typ, notnull, keys in rest:
                tags = " ".join(f"<<{k}>>" for k in keys.split(",") if k)
                star = "* " if notnull else ""
                lines.append(f"  {star}{name} : {typ} {tags}".rstrip())
            lines.append("}")
            lines.append("")
        # Tabel induk di kelompok lain: kotak ringkas tanpa kolom.
        outside = sorted({parent for child, parent in links if child in tables and parent not in tables})
        for o in outside:
            lines.append(f'entity "{o}" as {o} #EEEEEE')
        if outside:
            lines.append("")
        for child, parent in sorted(links):
            if child in tables and (parent in tables or parent in outside):
                lines.append(f"{parent} ||--o{{ {child}")
        lines += ["", "@enduml", ""]
    with open(OUTPUT, "w", encoding="utf-8", newline="\n") as f:
        f.write("\n".join(lines))
    print(f"{OUTPUT}: {len(columns)} tabel, {len(links)} hubungan, sampai migrasi {migration}")


if __name__ == "__main__":
    main()
