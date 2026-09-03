#!/usr/bin/env python3
"""
OVAV Curriculum Engine — curriculum_engine.py
===============================================
Toma los gaps del detector + perfil del estudiante y genera un learning path
con módulos ordenados por grafo de prerrequisitos.

Pipeline: load_gaps → load_modules → filter_completed → topological_sort →
          calculate_critical_path → estimate_hours → build_yaml

Spec canónica: .ovav/plan/education_roadmap.yaml (líneas 203-508)
Autor spec: Felipe + Beatriz (Valeria squad)
Implementación: Thavren (Platform Engineering)
"""

from __future__ import annotations

import argparse
import json
import sys
from collections import defaultdict, deque
from copy import deepcopy
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any

import yaml

# ═══════════════════════════════════════════════════════════════════════════
# Module Definitions — Prerequisite DAG
# ═══════════════════════════════════════════════════════════════════════════
# Fuente: education_roadmap.yaml + expansión Valeria (24 módulos total)
# Cada módulo define: id, name, skill, target_level, hours, prerequisites,
# hard (bool), topics, resources, mastery_criteria

MODULES: list[dict[str, Any]] = [
    # ── FUNDACIONALES COMPARTIDOS ───────────────────────────────────────
    {
        "id": "MOD-PY-01",
        "name": "Fundamentos de Programación con Python",
        "skill": "python",
        "target_level": "beginner",
        "hours": 40,
        "prerequisites": [],
        "hard": True,
        "topics": [
            "Variables, tipos de datos, operadores",
            "Estructuras de control (if, for, while)",
            "Funciones y scope",
            "Manejo de errores con try/except",
            "Módulos y paquetes",
        ],
        "resources": [
            {"type": "video", "title": "Python Crash Course — Google", "url": "https://developers.google.com/edu/python", "estimated_minutes": 480},
            {"type": "exercise", "title": "Exercism Python Track", "url": "https://exercism.org/tracks/python", "estimated_minutes": 600},
            {"type": "project", "title": "CLI Task Manager", "url": None, "estimated_minutes": 300},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Construir un gestor de tareas CLI con persistencia en archivo.",
        },
    },
    {
        "id": "MOD-PY-02",
        "name": "Python Intermedio: Estructuras de Datos y Algoritmos",
        "skill": "python",
        "target_level": "intermediate",
        "hours": 50,
        "prerequisites": ["MOD-PY-01"],
        "hard": True,
        "topics": [
            "Listas, tuplas, diccionarios, sets (complejidad)",
            "Comprensiones de lista y generadores",
            "Algoritmos de ordenamiento y búsqueda",
            "Recursión y memoización",
            "Testing con pytest",
        ],
        "resources": [
            {"type": "reading", "title": "Problem Solving with Algorithms — Python", "url": "https://runestone.academy/ns/books/published/pythonds/index.html", "estimated_minutes": 900},
            {"type": "exercise", "title": "LeetCode Easy Collection", "url": "https://leetcode.com/explore/featured/card/top-interview-questions-easy/", "estimated_minutes": 600},
        ],
        "mastery_criteria": {
            "assessment_type": "quiz",
            "passing_threshold": 0.8,
            "description": "Resolver 20 ejercicios LeetCode Easy con tests automatizados.",
        },
    },
    {
        "id": "MOD-GIT-01",
        "name": "Control de Versiones con Git",
        "skill": "git",
        "target_level": "intermediate",
        "hours": 20,
        "prerequisites": [],
        "hard": True,
        "topics": [
            "init, clone, add, commit, push, pull",
            "Branching (feature, release, hotfix)",
            "Merge, rebase, resolución de conflictos",
            "Pull requests y code review via GitHub",
            "Git hooks y conventional commits",
        ],
        "resources": [
            {"type": "reading", "title": "Pro Git Book — Scott Chacon", "url": "https://git-scm.com/book/en/v2", "estimated_minutes": 480},
            {"type": "exercise", "title": "Learn Git Branching", "url": "https://learngitbranching.js.org/", "estimated_minutes": 180},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Resolver 5 escenarios de merge conflict en un repo simulado.",
        },
    },
    {
        "id": "MOD-SQL-01",
        "name": "Bases de Datos Relacionales y SQL Fundamental",
        "skill": "sql",
        "target_level": "beginner",
        "hours": 30,
        "prerequisites": [],
        "hard": True,
        "topics": [
            "SELECT, WHERE, ORDER BY, LIMIT",
            "JOINs (INNER, LEFT, RIGHT, FULL)",
            "GROUP BY, HAVING, funciones de agregación",
            "Subconsultas y CTEs (WITH)",
            "Diseño de esquemas: PK, FK, índices, normalización",
        ],
        "resources": [
            {"type": "reading", "title": "SQL Tutorial — Mode Analytics", "url": "https://mode.com/sql-tutorial/", "estimated_minutes": 600},
            {"type": "exercise", "title": "SQLZoo Tutorials", "url": "https://sqlzoo.net/", "estimated_minutes": 480},
        ],
        "mastery_criteria": {
            "assessment_type": "quiz",
            "passing_threshold": 0.75,
            "description": "20 consultas SQL con complejidad creciente sobre dataset público.",
        },
    },
    {
        "id": "MOD-SQL-02",
        "name": "SQL Avanzado: Optimización y Window Functions",
        "skill": "sql",
        "target_level": "intermediate",
        "hours": 35,
        "prerequisites": ["MOD-SQL-01"],
        "hard": True,
        "topics": [
            "Window functions (ROW_NUMBER, RANK, LAG, LEAD)",
            "Optimización de queries con EXPLAIN ANALYZE",
            "Índices compuestos y partial indexes",
            "Transactions y niveles de aislamiento",
            "Vistas materializadas y particionamiento",
        ],
        "resources": [
            {"type": "reading", "title": "Use the Index, Luke!", "url": "https://use-the-index-luke.com/", "estimated_minutes": 600},
            {"type": "exercise", "title": "PostgreSQL Exercises", "url": "https://pgexercises.com/", "estimated_minutes": 480},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Optimizar 5 queries lentas con EXPLAIN ANALYZE. Reducir tiempo ≥50%.",
        },
    },
    # ── DATA SCIENCE PATH ────────────────────────────────────────────────
    {
        "id": "MOD-DS-01",
        "name": "Python para Ciencia de Datos: NumPy + Pandas",
        "skill": "python",
        "target_level": "advanced",
        "hours": 45,
        "prerequisites": ["MOD-PY-02"],
        "hard": True,
        "topics": [
            "Arrays NumPy: creación, slicing, broadcasting",
            "DataFrames Pandas: carga, limpieza, transformación",
            "Groupby, merge, pivot tables",
            "Series temporales con Pandas",
            "Optimización de memoria en datasets grandes",
        ],
        "resources": [
            {"type": "video", "title": "Data Analysis with Python — freeCodeCamp", "url": "https://www.youtube.com/watch?v=r-uOLxNrNk8", "estimated_minutes": 600},
            {"type": "exercise", "title": "Kaggle Pandas Micro-Course", "url": "https://www.kaggle.com/learn/pandas", "estimated_minutes": 240},
            {"type": "project", "title": "Limpieza y análisis de dataset real (>100K registros)", "url": None, "estimated_minutes": 600},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Cargar, limpiar y generar 5 visualizaciones estadísticas de un dataset público.",
        },
    },
    {
        "id": "MOD-STAT-01",
        "name": "Estadística Fundamental para Ciencia de Datos",
        "skill": "statistics",
        "target_level": "intermediate",
        "hours": 60,
        "prerequisites": ["MOD-PY-01"],
        "hard": False,
        "soft_prerequisites": ["MOD-PY-02"],
        "topics": [
            "Estadística descriptiva (media, mediana, varianza, percentiles)",
            "Distribuciones de probabilidad (normal, binomial, Poisson)",
            "Teorema del límite central",
            "Intervalos de confianza y pruebas de hipótesis",
            "Correlación y regresión lineal simple",
        ],
        "resources": [
            {"type": "reading", "title": "OpenIntro Statistics — 4th Edition", "url": "https://www.openintro.org/book/os/", "estimated_minutes": 1200},
            {"type": "exercise", "title": "StatLab Exercises (Python)", "url": None, "estimated_minutes": 600},
        ],
        "mastery_criteria": {
            "assessment_type": "quiz",
            "passing_threshold": 0.75,
            "description": "Prueba de 30 preguntas cubriendo inferencia estadística básica.",
        },
    },
    {
        "id": "MOD-ML-01",
        "name": "Machine Learning: Modelos Supervisados",
        "skill": "machine_learning",
        "target_level": "intermediate",
        "hours": 70,
        "prerequisites": ["MOD-DS-01", "MOD-STAT-01"],
        "hard": True,
        "topics": [
            "Regresión lineal y logística (scikit-learn)",
            "Árboles de decisión y Random Forest",
            "SVM y k-NN",
            "Validación cruzada y métricas de evaluación",
            "Feature engineering y selección de variables",
            "Overfitting, underfitting, regularización",
        ],
        "resources": [
            {"type": "video", "title": "ML Specialization — Andrew Ng", "url": "https://www.coursera.org/specializations/machine-learning-introduction", "estimated_minutes": 1800},
            {"type": "exercise", "title": "Kaggle ML Competitions", "url": "https://www.kaggle.com/competitions", "estimated_minutes": 900},
            {"type": "project", "title": "Modelo de predicción end-to-end con dataset real", "url": None, "estimated_minutes": 1200},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Entrenar y evaluar 3 modelos sobre dataset real. Notebook documentado.",
        },
    },
    {
        "id": "MOD-DV-01",
        "name": "Visualización de Datos Profesional",
        "skill": "data_visualization",
        "target_level": "intermediate",
        "hours": 35,
        "prerequisites": ["MOD-DS-01"],
        "hard": False,
        "soft_prerequisites": ["MOD-STAT-01"],
        "topics": [
            "Gramática de gráficos: matplotlib + seaborn",
            "Dashboards interactivos: Plotly + Streamlit",
            "Visualización geoespacial: folium, geopandas",
            "Storytelling con datos: elegir el gráfico correcto",
            "Accessibility y color blindness en visualizaciones",
        ],
        "resources": [
            {"type": "reading", "title": "Fundamentals of Data Visualization — Claus Wilke", "url": "https://clauswilke.com/dataviz/", "estimated_minutes": 600},
            {"type": "project", "title": "Dashboard ejecutivo con Streamlit", "url": None, "estimated_minutes": 600},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Crear dashboard interactivo con 5 visualizaciones desde dataset público.",
        },
    },
    # ── GO LANGUAGE PATH ──────────────────────────────────────────────────
    {
        "id": "MOD-GO-01",
        "name": "Go Fundamentals: Sintaxis y Concurrencia",
        "skill": "go",
        "target_level": "beginner",
        "hours": 35,
        "prerequisites": ["MOD-PY-01"],
        "hard": True,
        "topics": [
            "Sintaxis Go: packages, imports, tipos, zero values",
            "Funciones, métodos, interfaces",
            "Goroutines y channels",
            "Manejo de errores idiomático (error interface)",
            "Testing con el paquete testing",
        ],
        "resources": [
            {"type": "reading", "title": "A Tour of Go", "url": "https://go.dev/tour/", "estimated_minutes": 240},
            {"type": "exercise", "title": "Exercism Go Track", "url": "https://exercism.org/tracks/go", "estimated_minutes": 600},
            {"type": "project", "title": "CLI tool en Go con flags y subcomandos", "url": None, "estimated_minutes": 480},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Construir CLI tool con 3 subcomandos, flags, y tests.",
        },
    },
    {
        "id": "MOD-GO-02",
        "name": "Go Avanzado: Concurrencia y Sistemas",
        "skill": "go",
        "target_level": "intermediate",
        "hours": 40,
        "prerequisites": ["MOD-GO-01"],
        "hard": True,
        "topics": [
            "Patrones de concurrencia: pipelines, fan-in, fan-out, context",
            "Select statement y timeouts",
            "sync package: Mutex, WaitGroup, Once, Pool",
            "Profiling con pprof y benchmarks",
            "net/http server desde cero",
        ],
        "resources": [
            {"type": "reading", "title": "Concurrency in Go — Cox-Buday", "url": None, "estimated_minutes": 900},
            {"type": "exercise", "title": "Go Concurrency Patterns — Go Blog", "url": "https://go.dev/blog/pipelines", "estimated_minutes": 300},
            {"type": "project", "title": "HTTP proxy con rate limiting", "url": None, "estimated_minutes": 600},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Construir HTTP server concurrente con middleware chain y ≥80% test coverage.",
        },
    },
    {
        "id": "MOD-GO-03",
        "name": "Go en Producción: Microservicios y Deploy",
        "skill": "go",
        "target_level": "advanced",
        "hours": 45,
        "prerequisites": ["MOD-GO-02", "MOD-SQL-01"],
        "hard": True,
        "topics": [
            "Arquitectura de microservicios en Go",
            "gRPC y Protocol Buffers",
            "Docker multi-stage builds para Go",
            "Observabilidad: structured logging (slog), tracing (OpenTelemetry)",
            "Graceful shutdown y health checks",
        ],
        "resources": [
            {"type": "reading", "title": "Let's Go Further — Alex Edwards", "url": None, "estimated_minutes": 1200},
            {"type": "project", "title": "Microservicio Go con gRPC + REST gateway", "url": None, "estimated_minutes": 900},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Construir microservicio Go con gRPC, health checks, graceful shutdown y Docker deploy.",
        },
    },
    # ── BACKEND DEV PATH ──────────────────────────────────────────────────
    {
        "id": "MOD-BE-01",
        "name": "Diseño de APIs RESTful",
        "skill": "backend_development",
        "target_level": "intermediate",
        "hours": 40,
        "prerequisites": ["MOD-PY-02", "MOD-SQL-01"],
        "hard": True,
        "topics": [
            "Principios REST: recursos, verbos HTTP, códigos de estado",
            "Frameworks: FastAPI (Python), Gin (Go)",
            "Autenticación: JWT, OAuth2, API keys",
            "Validación de datos con Pydantic v2",
            "Documentación automática con OpenAPI/Swagger",
            "Rate limiting, CORS, paginación",
        ],
        "resources": [
            {"type": "reading", "title": "FastAPI Documentation", "url": "https://fastapi.tiangolo.com/", "estimated_minutes": 480},
            {"type": "project", "title": "API REST completa con CRUD + auth", "url": None, "estimated_minutes": 900},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.85,
            "description": "Construir API REST con 5 recursos, JWT auth, paginación y ≥80% test coverage.",
        },
    },
    {
        "id": "MOD-BE-02",
        "name": "Arquitectura de Backend Escalable",
        "skill": "backend_development",
        "target_level": "advanced",
        "hours": 50,
        "prerequisites": ["MOD-BE-01"],
        "hard": True,
        "topics": [
            "Microservicios vs monolitos modulares",
            "Message queues: RabbitMQ, Redis Pub/Sub",
            "Caching strategies: Redis, CDN, in-memory",
            "Database scaling: read replicas, sharding",
            "Observabilidad: structured logging, tracing, metrics",
            "Event-driven architecture y CQRS",
        ],
        "resources": [
            {"type": "reading", "title": "Designing Data-Intensive Applications — Kleppmann", "url": None, "estimated_minutes": 1800},
            {"type": "project", "title": "Sistema de 3 microservicios con message queue", "url": None, "estimated_minutes": 1200},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Diseñar y construir 3 microservicios comunicándose via Redis. Docker Compose.",
        },
    },
    # ── FRONTEND DEV PATH ─────────────────────────────────────────────────
    {
        "id": "MOD-FE-01",
        "name": "Desarrollo Frontend Moderno con React + TypeScript",
        "skill": "frontend_development",
        "target_level": "intermediate",
        "hours": 50,
        "prerequisites": ["MOD-PY-01"],  # programming fundamentals transfer
        "hard": True,
        "topics": [
            "JavaScript/TypeScript: tipos, async/await, módulos ES",
            "React: componentes, hooks, estado, efectos",
            "CSS moderno: Flexbox, Grid, Tailwind",
            "Routing con React Router",
            "State management: Context API, Zustand",
            "Testing: Vitest + React Testing Library",
        ],
        "resources": [
            {"type": "reading", "title": "React Documentation (beta)", "url": "https://react.dev/", "estimated_minutes": 600},
            {"type": "exercise", "title": "Frontend Mentor Challenges", "url": "https://www.frontendmentor.io/", "estimated_minutes": 480},
            {"type": "project", "title": "Dashboard SPA con React + TypeScript", "url": None, "estimated_minutes": 900},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Construir SPA con 3 vistas, API integration, responsive design y ≥80% coverage.",
        },
    },
    # ── DEVOPS PATH ───────────────────────────────────────────────────────
    {
        "id": "MOD-DO-01",
        "name": "Contenedores y Orquestación con Docker",
        "skill": "devops_and_cloud",
        "target_level": "intermediate",
        "hours": 30,
        "prerequisites": ["MOD-PY-01", "MOD-GIT-01"],
        "hard": True,
        "topics": [
            "Dockerfiles multi-stage, optimización de capas",
            "Docker Compose para entornos multi-servicio",
            "Volúmenes, redes, secretos en Docker",
            "Registros: Docker Hub, GHCR",
            "Seguridad: non-root user, read-only FS, capability drop",
        ],
        "resources": [
            {"type": "reading", "title": "Docker Documentation", "url": "https://docs.docker.com/", "estimated_minutes": 480},
            {"type": "exercise", "title": "Docker Mastery Course", "url": None, "estimated_minutes": 600},
            {"type": "project", "title": "Dockerizar aplicación completa con CI/CD", "url": None, "estimated_minutes": 480},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Dockerizar una app Python + PostgreSQL con CI/CD en GitHub Actions.",
        },
    },
    # ── SYSTEM DESIGN PATH ────────────────────────────────────────────────
    {
        "id": "MOD-SD-01",
        "name": "Diseño de Sistemas Distribuidos",
        "skill": "system_design",
        "target_level": "intermediate",
        "hours": 45,
        "prerequisites": ["MOD-BE-01", "MOD-SQL-02"],
        "hard": True,
        "topics": [
            "System design interview framework (requirements → estimation → design)",
            "Load balancers, reverse proxies, CDNs",
            "Database choices: SQL vs NoSQL vs NewSQL",
            "CAP theorem y consistency patterns",
            "Design patterns: circuit breaker, bulkhead, retry, backoff",
        ],
        "resources": [
            {"type": "reading", "title": "System Design Interview — Alex Xu", "url": None, "estimated_minutes": 900},
            {"type": "exercise", "title": "Design practice: URL shortener, chat app, news feed", "url": None, "estimated_minutes": 600},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Design doc completo para sistema de 1M+ usuarios con diagramas y trade-offs.",
        },
    },
    # ── SECURITY PATH ─────────────────────────────────────────────────────
    {
        "id": "MOD-SEC-01",
        "name": "Ciberseguridad Aplicada para Developers",
        "skill": "cybersecurity",
        "target_level": "intermediate",
        "hours": 35,
        "prerequisites": ["MOD-BE-01"],
        "hard": False,
        "soft_prerequisites": ["MOD-DO-01"],
        "topics": [
            "OWASP Top 10: detección y mitigación",
            "SQL injection, XSS, CSRF, SSRF prácticos",
            "Criptografía aplicada: hashing, AES, RSA, TLS",
            "Modelado de amenazas: STRIDE, attack trees",
            "Secret management: Vault, environment variables, .env",
        ],
        "resources": [
            {"type": "reading", "title": "OWASP Top 10 — 2025", "url": "https://owasp.org/www-project-top-ten/", "estimated_minutes": 300},
            {"type": "exercise", "title": "OWASP WebGoat", "url": "https://owasp.org/www-project-webgoat/", "estimated_minutes": 480},
            {"type": "project", "title": "Security audit de aplicación existente", "url": None, "estimated_minutes": 600},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Realizar security audit con ≥5 vulnerabilidades encontradas y mitigadas.",
        },
    },
    # ── PROFESSIONAL SKILLS ───────────────────────────────────────────────
    {
        "id": "MOD-TW-01",
        "name": "Technical Writing para Ingenieros",
        "skill": "technical_writing",
        "target_level": "intermediate",
        "hours": 20,
        "prerequisites": [],
        "hard": False,
        "topics": [
            "Estructura de documentos técnicos (ADR, RFC, postmortem)",
            "Claridad: voz activa, frases cortas, ejemplos concretos",
            "Diagramas: Mermaid, PlantUML, Excalidraw",
            "README-driven development",
            "Documentación de API con OpenAPI",
        ],
        "resources": [
            {"type": "reading", "title": "Google Technical Writing Courses", "url": "https://developers.google.com/tech-writing", "estimated_minutes": 240},
            {"type": "project", "title": "ADR documentado para decisión arquitectónica real", "url": None, "estimated_minutes": 180},
        ],
        "mastery_criteria": {
            "assessment_type": "peer_review",
            "passing_threshold": 0.8,
            "description": "Escribir un ADR de 500+ palabras revisado y aprobado por peer.",
        },
    },
    {
        "id": "MOD-CR-01",
        "name": "Code Review Efectiva",
        "skill": "code_review",
        "target_level": "intermediate",
        "hours": 15,
        "prerequisites": ["MOD-PY-02", "MOD-GIT-01"],
        "hard": False,
        "topics": [
            "Qué buscar: bugs, seguridad, performance, estilo",
            "Feedback constructivo: preguntas, no órdenes",
            "Conventional comments: praise, suggestion, nitpick, issue",
            "Automatización: linters, formateadores, CI checks",
            "Code review culture: velocidad, tamaño de PRs, ownership",
        ],
        "resources": [
            {"type": "reading", "title": "Google Code Review Guidelines", "url": "https://google.github.io/eng-practices/review/", "estimated_minutes": 180},
            {"type": "exercise", "title": "Code Review practice: 5 PRs simulados", "url": None, "estimated_minutes": 240},
        ],
        "mastery_criteria": {
            "assessment_type": "peer_review",
            "passing_threshold": 0.8,
            "description": "Realizar code review de 3 PRs reales con feedback accionable documentado.",
        },
    },
    {
        "id": "MOD-PS-01",
        "name": "Problem Solving Avanzado",
        "skill": "problem_solving",
        "target_level": "intermediate",
        "hours": 25,
        "prerequisites": ["MOD-PY-02"],
        "hard": False,
        "topics": [
            "Descomposición de problemas complejos",
            "Pensamiento algorítmico: dividir y conquistar, DP, greedy",
            "Depuración sistemática: binary search debugging, rubber ducking",
            "Trade-off analysis: tiempo vs espacio vs simplicidad",
            "Comunicación de soluciones técnicas a audiencias no técnicas",
        ],
        "resources": [
            {"type": "reading", "title": "The Algorithm Design Manual — Skiena", "url": None, "estimated_minutes": 900},
            {"type": "exercise", "title": "Advent of Code (10 días)", "url": "https://adventofcode.com/", "estimated_minutes": 600},
        ],
        "mastery_criteria": {
            "assessment_type": "project",
            "passing_threshold": 0.8,
            "description": "Resolver 10 problemas de Advent of Code documentando approach y trade-offs.",
        },
    },
]


# ═══════════════════════════════════════════════════════════════════════════
# Core Engine
# ═══════════════════════════════════════════════════════════════════════════

LEVEL_ORDER: dict[str, int] = {
    "beginner": 0,
    "novice": 1,
    "intermediate": 2,
    "advanced": 3,
    "expert": 4,
}


class CurriculumEngine:
    """Generador de learning paths basado en gaps y grafo de prerrequisitos."""

    def __init__(self, gaps_json: dict[str, Any], profile_yaml: dict[str, Any]):
        self.gaps = gaps_json
        self.profile = profile_yaml
        self.student_id = ""
        self.career_goal = ""
        self.available_hours_per_week = 10
        self.time_constraint_days = 0
        self.prior_modules: list[str] = []
        self.learning_style = "mixed"
        self.preferred_language = "es"

        self.module_map: dict[str, dict[str, Any]] = {}
        self.selected_modules: list[dict[str, Any]] = []
        self.warnings: list[str] = []

        self._parse_inputs()

    def _parse_inputs(self) -> None:
        """Validar y extraer datos de entrada."""
        # Parse gaps
        if "student_id" not in self.gaps:
            raise ValueError("gaps JSON requiere 'student_id'")
        self.student_id = str(self.gaps["student_id"])
        self.career_goal = str(self.gaps.get("career_goal", ""))

        # Parse profile
        profile = self.profile.get("profile", self.profile)
        if "student_id" not in profile:
            raise ValueError("profile YAML requiere 'student_id'")
        if self.student_id != str(profile["student_id"]):
            self.warnings.append(
                f"student_id mismatch: gaps={self.student_id}, "
                f"profile={profile['student_id']}. Usando gaps."
            )

        self.available_hours_per_week = int(profile.get("available_hours_per_week", 10))
        self.time_constraint_days = int(profile.get("time_constraint_days", 0))
        self.prior_modules = list(profile.get("prior_modules_completed", []))
        self.learning_style = str(profile.get("learning_style", "mixed"))
        self.preferred_language = str(profile.get("preferred_language", "es"))

        # Build module map
        for mod in MODULES:
            self.module_map[mod["id"]] = mod

    # ── Main Pipeline ────────────────────────────────────────────────────

    def run(self) -> dict[str, Any]:
        """Ejecutar el pipeline completo y retornar YAML como dict."""
        self._select_modules_for_gaps()
        self._filter_completed()
        self._topological_sort()
        self._assign_orders()
        total_hours = self._calculate_total_hours()
        critical_path_hours = self._calculate_critical_path()
        self._check_time_constraint(total_hours)

        return self._build_output(total_hours, critical_path_hours)

    def _select_modules_for_gaps(self) -> None:
        """
        Seleccionar módulos necesarios para cerrar los gaps.
        Para cada gap, buscar módulos que enseñen esa skill hasta target_level.
        """
        gap_list = self.gaps.get("gaps", [])
        needed_skills: dict[str, str] = {}  # skill → target_level

        for gap in gap_list:
            skill = gap["skill"]
            target = gap["target_level"]
            current = gap.get("estimated_level", "beginner")

            # Mantener el target más alto si ya hay registro
            if skill in needed_skills:
                current_target_idx = LEVEL_ORDER.get(needed_skills[skill], 2)
                new_target_idx = LEVEL_ORDER.get(target, 2)
                if new_target_idx > current_target_idx:
                    needed_skills[skill] = target
            else:
                needed_skills[skill] = target

        # Seleccionar módulos que enseñan esas skills
        selected_ids: set[str] = set()
        for mod in MODULES:
            skill = mod["skill"]
            if skill in needed_skills:
                mod_target_idx = LEVEL_ORDER.get(mod["target_level"], 2)
                required_idx = LEVEL_ORDER.get(needed_skills[skill], 2)
                if mod_target_idx <= required_idx:
                    selected_ids.add(mod["id"])
                    # También agregar prerrequisitos
                    self._add_prerequisites(mod["id"], selected_ids)

        self.selected_modules = [
            deepcopy(self.module_map[mid])
            for mid in selected_ids
            if mid in self.module_map
        ]

    def _add_prerequisites(self, module_id: str, selected: set[str]) -> None:
        """Agregar recursivamente prerrequisitos a la selección."""
        if module_id not in self.module_map:
            return
        mod = self.module_map[module_id]
        for prereq_id in mod.get("prerequisites", []):
            if prereq_id not in selected:
                selected.add(prereq_id)
                self._add_prerequisites(prereq_id, selected)

    def _filter_completed(self) -> None:
        """Omitir módulos cuyo target_level ya fue alcanzado."""
        # Si el estudiante ya completó módulos, omitirlos
        completed = set(self.prior_modules)

        # También omitir si el nivel de la skill ya es ≥ target del módulo
        gap_list = self.gaps.get("gaps", [])
        skill_levels: dict[str, int] = {}
        for gap in gap_list:
            skill = gap["skill"]
            estimated_idx = LEVEL_ORDER.get(gap.get("estimated_level", "beginner"), 0)
            skill_levels[skill] = estimated_idx

        filtered = []
        for mod in self.selected_modules:
            if mod["id"] in completed:
                continue
            skill = mod["skill"]
            target_idx = LEVEL_ORDER.get(mod["target_level"], 2)
            current_idx = skill_levels.get(skill, 0)
            if current_idx >= target_idx:
                continue  # Ya tiene el nivel necesario
            filtered.append(mod)

        self.selected_modules = filtered

    def _topological_sort(self) -> None:
        """
        Ordenar módulos respetando prerrequisitos (Kahn's algorithm).
        Si hay ciclo, fallar con error.
        """
        # Build adjacency and in-degree
        selected_ids = {m["id"] for m in self.selected_modules}
        adj: dict[str, list[str]] = defaultdict(list)
        in_degree: dict[str, int] = {m["id"]: 0 for m in self.selected_modules}

        for mod in self.selected_modules:
            for prereq_id in mod.get("prerequisites", []):
                if prereq_id in selected_ids and mod["id"] in selected_ids:
                    adj[prereq_id].append(mod["id"])
                    in_degree[mod["id"]] += 1

        # Kahn's algorithm
        queue: deque[str] = deque(
            mid for mid in selected_ids if in_degree.get(mid, 0) == 0
        )
        sorted_ids: list[str] = []

        while queue:
            node = queue.popleft()
            sorted_ids.append(node)
            for neighbor in adj[node]:
                in_degree[neighbor] -= 1
                if in_degree[neighbor] == 0:
                    queue.append(neighbor)

        if len(sorted_ids) != len(selected_ids):
            remaining = selected_ids - set(sorted_ids)
            raise ValueError(
                f"Ciclo detectado en el grafo de prerrequisitos. "
                f"Módulos no resueltos: {remaining}"
            )

        # Reordenar self.selected_modules según sorted_ids
        order_map = {mid: i for i, mid in enumerate(sorted_ids)}
        self.selected_modules.sort(key=lambda m: order_map.get(m["id"], 999))

    def _assign_orders(self) -> None:
        """Asignar order secuencial a cada módulo."""
        for i, mod in enumerate(self.selected_modules):
            mod["order"] = i + 1

    def _calculate_total_hours(self) -> int:
        """Sumar horas de todos los módulos seleccionados."""
        return sum(m.get("hours", 0) for m in self.selected_modules)

    def _calculate_critical_path(self) -> int:
        """
        Calcular el camino crítico (longest path) en el DAG de módulos.
        DP: longest_path[node] = hours[node] + max(longest_path[suc])
        """
        selected_ids = {m["id"] for m in self.selected_modules}
        hours_map = {m["id"]: m.get("hours", 0) for m in self.selected_modules}

        # Build reverse adjacency (sucesores)
        succ: dict[str, list[str]] = defaultdict(list)
        for mod in self.selected_modules:
            for prereq_id in mod.get("prerequisites", []):
                if prereq_id in selected_ids:
                    succ[prereq_id].append(mod["id"])

        # Topological order (already computed in _topological_sort)
        topo_order = [m["id"] for m in self.selected_modules]

        # DP bottom-up
        longest: dict[str, int] = {}
        max_path = 0
        for node in reversed(topo_order):
            best_succ = 0
            for s in succ.get(node, []):
                best_succ = max(best_succ, longest.get(s, 0))
            longest[node] = hours_map.get(node, 0) + best_succ
            max_path = max(max_path, longest[node])

        return max_path

    def _check_time_constraint(self, total_hours: int) -> None:
        """Verificar si el plan excede el time constraint."""
        if self.time_constraint_days <= 0:
            return

        # Días necesarios = total_hours / (available_hours_per_week / 7)
        daily_hours = self.available_hours_per_week / 7.0
        if daily_hours <= 0:
            self.warnings.append(
                "available_hours_per_week es 0. No se puede estimar duración."
            )
            return

        needed_days = math.ceil(total_hours / daily_hours)
        if needed_days > self.time_constraint_days:
            self.warnings.append(
                f"Plan excede time constraint: necesita {needed_days} días "
                f"({total_hours}h a {self.available_hours_per_week}h/semana) "
                f"pero el límite es {self.time_constraint_days} días."
            )

    def _build_output(self, total_hours: int, critical_path_hours: int) -> dict[str, Any]:
        """Construir output YAML como dict."""
        now_iso = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S%z")
        now_iso = now_iso[:-2] + ":" + now_iso[-2:]

        modules_output = []
        for mod in self.selected_modules:
            m = {
                "id": mod["id"],
                "name": mod["name"],
                "skill": mod["skill"],
                "target_level": mod["target_level"],
                "hours": mod["hours"],
                "prerequisites": mod.get("prerequisites", []),
                "order": mod.get("order", 0),
                "topics": mod.get("topics", []),
                "resources": mod.get("resources", []),
                "mastery_criteria": mod.get("mastery_criteria", {}),
            }
            # Keep soft prerequisites if present
            if "soft_prerequisites" in mod:
                m["soft_prerequisites"] = mod["soft_prerequisites"]
            modules_output.append(m)

        output: dict[str, Any] = {
            "student_id": self.student_id,
            "career_goal": self.career_goal,
            "generated_at": now_iso,
            "total_estimated_hours": total_hours,
            "total_modules": len(self.selected_modules),
            "critical_path_hours": critical_path_hours,
            "learning_style": self.learning_style,
            "preferred_language": self.preferred_language,
            "modules": modules_output,
            "warnings": self.warnings if self.warnings else [],
            "metadata": {
                "engine_version": "1.0.0",
                "available_hours_per_week": self.available_hours_per_week,
                "time_constraint_days": self.time_constraint_days,
                "prior_modules_completed": self.prior_modules,
            },
        }
        return output


# ═══════════════════════════════════════════════════════════════════════════
# CLI
# ═══════════════════════════════════════════════════════════════════════════

import math


def main() -> None:
    parser = argparse.ArgumentParser(
        description="OVAV Curriculum Engine — Gaps → Learning Path"
    )
    parser.add_argument(
        "gaps_file",
        help="JSON file con output de gap_detector.py",
    )
    parser.add_argument(
        "profile_file",
        help="YAML file con perfil del estudiante",
    )
    parser.add_argument(
        "--json-output", action="store_true",
        help="Emitir JSON en lugar de YAML"
    )
    args = parser.parse_args()

    try:
        with open(args.gaps_file) as f:
            gaps = json.load(f)
        with open(args.profile_file) as f:
            profile = yaml.safe_load(f)

        if profile is None:
            print(json.dumps({"error": "Profile YAML vacío o inválido"}))
            sys.exit(1)

        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        if args.json_output:
            print(json.dumps(result, indent=2, ensure_ascii=False))
        else:
            print(yaml.dump(result, allow_unicode=True, sort_keys=False, default_flow_style=False))

    except ValueError as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(2)
    except (json.JSONDecodeError, yaml.YAMLError) as e:
        print(json.dumps({"error": f"Parse error: {e}"}))
        sys.exit(2)


if __name__ == "__main__":
    main()
