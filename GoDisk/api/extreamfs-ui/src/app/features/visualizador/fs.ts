import { Component, OnInit, NgZone, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, ParamMap, RouterLink } from '@angular/router';
import { combineLatest } from 'rxjs';
import { finalize } from 'rxjs/operators';

import { FsService, FsFindResp } from '../../core/services/fs';
import { LSReport, LSItem } from '../../core/services/reports';

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

  // Estado del visor de archivo
  filePath: string | null = null;
  fileText: string | null = null;
  fileLoading = false;
  fileError = '';

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

          // Al cambiar de carpeta, cerramos el visor de archivo
          this.filePath = null;
          this.fileText = null;
          this.fileError = '';
          this.fileLoading = false;

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

  // Abrir archivo (si es texto) en el visor
  openFile(f: FileEnt): void {
    if (!this.id || !f?.abs) return;

    this.zone.run(() => {
      this.filePath = f.abs;
      this.fileText = null;
      this.fileError = '';
      this.fileLoading = true;
      this.cdr.detectChanges();
    });

    this.fs.fileText(this.id, f.abs)
      .pipe(finalize(() => {
        this.zone.run(() => {
          this.fileLoading = false;
          this.cdr.detectChanges();
        });
      }))
      .subscribe({
        next: (txt: string) => this.zone.run(() => {

          const isProbablyText = !/[\x00-\x08\x0E-\x1F]/.test(txt);
          if (isProbablyText) {
            this.fileText = txt;
          } else {
            this.fileError = 'El archivo no parece ser de texto.';
          }
          this.cdr.detectChanges();
        }),
        error: (err) => this.zone.run(() => {
          this.fileError =
            (err?.error && typeof err.error === 'object' && err.error.error) ||
            (typeof err?.error === 'string' ? err.error : '') ||
            err?.message ||
            'No se pudo leer el archivo.';
          this.cdr.detectChanges();
        })
      });
  }

  closeFile(): void {
    this.zone.run(() => {
      this.filePath = null;
      this.fileText = null;
      this.fileError = '';
      this.fileLoading = false;
      this.cdr.detectChanges();
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
