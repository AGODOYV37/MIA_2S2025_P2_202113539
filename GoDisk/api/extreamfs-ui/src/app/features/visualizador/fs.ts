// GoDisk/api/extreamfs-ui/src/app/features/visualizador/fs.ts
import { Component, OnInit, NgZone, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, ParamMap, RouterLink } from '@angular/router';
import { combineLatest } from 'rxjs';
import { finalize } from 'rxjs/operators';

import { FsService, FsFindResp } from '../../core/services/fs';

type DirEnt = { name: string; abs: string };
type FileEnt = { name: string; abs: string };

@Component({
  selector: 'app-fs-explorer',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './fs.html',
  styleUrls: ['./fs.scss'],
})
export class FsExplorerComponent implements OnInit {
  id = '';
  ruta = '/';

  loading = true;
  error = '';

  dirs: DirEnt[] = [];
  files: FileEnt[] = [];

  constructor(
    private route: ActivatedRoute,
    private fs: FsService,
    private zone: NgZone,
    private cdr: ChangeDetectorRef
  ) {}

  get isRoot(): boolean {
    return this.ruta === '/';
  }

  ngOnInit(): void {
    // Reacciona a cambios de :id y ?ruta=
    combineLatest([this.route.paramMap, this.route.queryParamMap]).subscribe(
      ([pm, qm]: [ParamMap, ParamMap]) => {
        const newId = (pm.get('id') || '').trim();
        let newRuta = (qm.get('ruta') || '/').trim();
        newRuta = this.normalizeRuta(newRuta);

        this.zone.run(() => {
          this.id = newId;
          this.ruta = newRuta;
          this.error = '';
          this.loading = true;
          this.cdr.detectChanges();
        });

        if (!this.id) {
          this.zone.run(() => {
            this.error = 'Falta el ID de partición montada.';
            this.loading = false;
            this.cdr.detectChanges();
          });
          return;
        }

        this.load();
      }
    );
  }

  private load(): void {
    // Usa /api/fs/find para construir la navegación (respeta permisos y sesión en el backend)
    this.fs
      .findList(this.id, this.ruta)
      .pipe(
        finalize(() =>
          this.zone.run(() => {
            this.loading = false;
            this.cdr.detectChanges();
          })
        )
      )
      .subscribe({
        next: (resp: FsFindResp) =>
          this.zone.run(() => {
            const baseAbs = this.ruta;

            this.dirs = (resp?.dirs ?? []).map((name) => ({
              name,
              abs: this.joinAbs(baseAbs, name),
            }));

            this.files = (resp?.files ?? []).map((name) => ({
              name,
              abs: this.joinAbs(baseAbs, name),
            }));

            this.error = '';
            this.cdr.detectChanges();
          }),
        error: (err) =>
          this.zone.run(() => {
            this.error =
              (err?.error && typeof err.error === 'object' && err.error.error) ||
              (typeof err?.error === 'string' ? err.error : '') ||
              err?.message ||
              'No se pudo listar la ruta.';
            this.dirs = [];
            this.files = [];
            this.cdr.detectChanges();
          }),
      });
  }

  // ========= Utils usados por la plantilla =========

  crumbs(): { label: string; abs: string }[] {
    const r = this.ruta;
    if (r === '/' || !r) return [{ label: '/', abs: '/' }];

    const parts = r.replace(/^\/+/, '').split('/').filter(Boolean);
    const out: { label: string; abs: string }[] = [{ label: '/', abs: '/' }];

    let acc = '';
    for (const p of parts) {
      acc += '/' + p;
      out.push({ label: p, abs: acc || '/' });
    }
    return out;
  }

  parent(p: string): string {
    const n = this.normalizeRuta(p);
    if (n === '/') return '/';
    const segs = n.split('/').filter(Boolean);
    segs.pop();
    const up = '/' + segs.join('/');
    return up === '' ? '/' : up;
  }

  // ========= Helpers internos =========

  private normalizeRuta(r: string): string {
    if (!r) return '/';
    let s = r.replace(/\\+/g, '/');
    s = s.replace(/\/{2,}/g, '/');
    if (!s.startsWith('/')) s = '/' + s;
    if (s.length > 1) s = s.replace(/\/+$/, '');
    return s;
  }

  private joinAbs(dir: string, name: string): string {
    const d = this.normalizeRuta(dir);
    const n = (name || '').replace(/^\/+/, '');
    const j = d === '/' ? `/${n}` : `${d}/${n}`;
    return this.normalizeRuta(j);
  }
}
