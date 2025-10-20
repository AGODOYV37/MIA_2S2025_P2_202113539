import { Component, OnInit, NgZone, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router} from '@angular/router';
import { combineLatest } from 'rxjs';
import { FsService, FindRes } from '../../core/services/fs';

type Child = { name: string; abs: string; type: 'dir'|'file' };

@Component({
  selector: 'app-fs-explorer',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './fs.html',
  styleUrls: ['./fs.scss']
})
export class FsExplorerComponent implements OnInit {
  id = '';
  ruta = '/';
  loading = true;
  error = '';
  // lo que mostramos (solo hijos inmediatos)
  dirs: Child[] = [];
  files: Child[] = [];

  constructor(
    private fs: FsService,
    private route: ActivatedRoute,
    private router: Router,
    private zone: NgZone,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit(): void {
    combineLatest([this.route.paramMap, this.route.queryParamMap]).subscribe(([pm, qm]) => {
      const id = (pm.get('id') || '').trim();
      const ruta = (qm.get('ruta') || '/').trim();
      this.zone.run(() => {
        this.id = id;
        this.ruta = this.normalizePath(ruta || '/');
        this.error = '';
        this.loading = true;
        this.dirs = [];
        this.files = [];
        this.cdr.detectChanges();
      });
      this.load();
    });
  }

  private load() {
    this.fs.find(this.ruta, '*').subscribe({
      next: (res: FindRes) => this.zone.run(() => {
        const children = this.immediateChildren(res.items, this.ruta);
        // separa carpetas/archivos y ordena
        this.dirs = children.filter(c => c.type === 'dir')
                            .sort((a,b) => a.name.localeCompare(b.name));
        this.files = children.filter(c => c.type === 'file')
                             .sort((a,b) => a.name.localeCompare(b.name));
        this.loading = false;
        this.cdr.detectChanges();
      }),
      error: (err) => this.zone.run(() => {
        this.error = (err?.error || err?.message || 'Error listando');
        this.loading = false;
        this.cdr.detectChanges();
      })
    });
  }

  // Devuelve hijos inmediatos bajo 'base'. Deducción de tipo por presencia de subrutas.
    private immediateChildren(all: string[], base: string): Child[] {
    const norm = (p: string) => this.normalizePath(p);
    base = norm(base);
    const prefix = base === '/' ? '/' : base + '/';

    const under = all.map(norm).filter(p => p !== base && p.startsWith(prefix));

    const seen = new Map<string, Child>();
    for (const abs of under) {
        const rel = abs.slice(prefix.length);
        if (!rel) continue;
        const first = rel.split('/')[0];
        if (!first) continue;
        const key = first.toLowerCase();

        // 👇 Carpeta SÓLO si hay algo más profundo con el mismo primer segmento
        const isDir = under.some(p => p.startsWith(prefix + first + '/'));
        const type: 'dir' | 'file' = isDir ? 'dir' : 'file';

        if (!seen.has(key)) {
        seen.set(key, { name: first, abs: this.join(base, first), type });
        }
    }
    return Array.from(seen.values());
    }

  // Navegación
  goTo(child: Child) {
    if (child.type !== 'dir') return; // por ahora archivos no se abren
    this.router.navigate(['/visualizador', this.id, 'fs'], {
      queryParams: { ruta: child.abs }
    });
  }

  // Breadcrumb: segmentos clicables
  crumbs(): { label: string; abs: string }[] {
    const parts = this.ruta === '/' ? [] : this.ruta.split('/').filter(Boolean);
    const acc: { label: string; abs: string }[] = [{ label: '/', abs: '/' }];
    let cur = '';
    for (const p of parts) {
      cur = this.join(cur || '/', p);
      acc.push({ label: p, abs: cur });
    }
    return acc;
  }

  goCrumb(abs: string) {
    this.router.navigate(['/visualizador', this.id, 'fs'], { queryParams: { ruta: abs }});
  }

    get isRoot(): boolean {
    return this.normalizePath(this.ruta) === '/';
  }

  private parentOf(p: string): string {
    p = this.normalizePath(p);
    if (p === '/') return '/';
    // quita trailing slash
    if (p.endsWith('/')) p = p.slice(0, -1);
    const cut = p.lastIndexOf('/');
    const parent = cut <= 0 ? '/' : p.slice(0, cut);
    return parent || '/';
  }

  goUp(): void {
    const parent = this.parentOf(this.ruta);

    if (parent === '/') {
      // si ya estás en raíz, vuelve a las particiones del disco
      this.router.navigate(['/visualizador', this.id]);
    } else {
      // mismo explorador, un nivel arriba
      this.router.navigate(
        ['/visualizador', this.id, 'fs'],
        { queryParams: { ruta: parent } }
      );
    }
  }


  private normalizePath(p: string): string {
    if (!p) return '/';
    p = p.replace(/\\+/g, '/').replace(/\/{2,}/g, '/').trim();
    if (!p.startsWith('/')) p = '/' + p;
    return p === '' ? '/' : p;
  }

  private join(parent: string, child: string): string {
    if (!parent || parent === '/') return `/${child}`;
    return `${parent}/${child}`;
  }
}
