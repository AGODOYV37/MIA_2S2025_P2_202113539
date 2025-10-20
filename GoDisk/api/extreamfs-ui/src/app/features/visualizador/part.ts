import { Component, OnInit, NgZone, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { finalize } from 'rxjs/operators';
import { forkJoin } from 'rxjs';

import { Reports, DiskReport } from '../../core/services/reports';
import { MountsService, MountView } from '../../core/services/mount';
import { AuthService } from '../../core/services/auth'; // ← NUEVO

type PartCard = {
  kind: 'P' | 'L';
  name: string;
  start: number;
  end: number;
  size: number;
  percent: number;
  mountId?: string;
};

@Component({
  selector: 'app-visualizador-part',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './part.html',
  styleUrls: ['./part.scss'],
})
export class VisualizadorPartComponent implements OnInit {
  id = '';
  loading = true;
  error = '';
  diskPath = '';
  parts: PartCard[] = [];

  constructor(
    private reports: Reports,
    private mountsSvc: MountsService,
    private route: ActivatedRoute,
    private zone: NgZone,
    private cdr: ChangeDetectorRef,
    public auth: AuthService,             // ← NUEVO
  ) {}

  ngOnInit(): void {
    this.route.paramMap.subscribe((pm) => {
      const id = (pm.get('id') || '').trim();
      this.zone.run(() => {
        this.id = id;
        this.error = '';
        this.parts = [];
        this.loading = true;
        this.cdr.detectChanges();
      });

      if (!id) {
        this.zone.run(() => {
          this.error = 'ID de partición requerido';
          this.loading = false;
          this.cdr.detectChanges();
        });
        return;
      }

      this.cargar(id);
    });
  }

  private cargar(id: string) {
    forkJoin({
      disk: this.reports.getDiskWithRetry(id, 2, 250),
      mounts: this.mountsSvc.getAll(),
    })
      .pipe(
        finalize(() =>
          this.zone.run(() => {
            this.loading = false;
            this.cdr.detectChanges();
          })
        )
      )
      .subscribe({
        next: ({ disk, mounts }) =>
          this.zone.run(() => {
            this.diskPath = disk?.diskPath || '';
            this.parts = this.buildParts(disk, mounts || []);
            if (!this.parts.length) {
              this.parts = this.buildPartsLoose(disk, mounts || []);
            }
            this.cdr.detectChanges();
          }),
        error: (err) =>
          this.zone.run(() => {
            this.error =
              (err?.error && typeof err.error === 'object' && err.error.error) ||
              (typeof err?.error === 'string' ? err.error : '') ||
              err?.message ||
              'Error cargando particiones';
            this.parts = [];
            this.cdr.detectChanges();
          }),
      });
  }

  baseName(p: string): string {
    if (!p) return '';
    const clean = p.replace(/\\+/g, '/');
    const parts = clean.split('/');
    return parts.pop() || clean;
  }

  private buildParts(rep: DiskReport, mounts: MountView[]): PartCard[] {
    const prim = (rep?.segments || []).filter((s: any) => s?.kind === 'P');
    const logi = (rep?.extended?.segments || []).filter((s: any) => s?.kind === 'L');

    const normPath = (p: string) => (p || '').replace(/\\+/g, '/').toLowerCase().trim();
    const normName = (s: string) => (s || '').replace(/\s+/g, '').toLowerCase().trim();

    const onThisDisk = (mounts || []).filter(
      (m) => normPath(m.diskPath) === normPath(rep?.diskPath || '')
    );

    const byStartSize = new Map<string, MountView>();
    for (const m of onThisDisk) {
      const key = `${Number(m.start)||0}|${Number(m.size)||0}`;
      byStartSize.set(key, m);
    }

    const byName = new Map<string, MountView>();
    for (const m of onThisDisk) {
      const k = normName(m.name || '');
      if (k) byName.set(k, m);
    }

    const fixWeird = (raw: string) => {
      const s = raw || '';
      const m1 = s.match(/^(.*?)(\d)\2$/);
      if (m1) return m1[1] + m1[2];
      const m2 = s.match(/^part1(\d+)$/i);
      if (m2) return 'Part' + m2[1];
      return s;
    };

    const findMountId = (seg: any): string | undefined => {
      const key = `${Number(seg.start)||0}|${Number(seg.size)||0}`;
      const hitByPos = byStartSize.get(key);
      if (hitByPos) return hitByPos.id;

      const candidates = [seg.name, fixWeird(seg.name)].map(normName).filter(Boolean);

      for (const c of candidates) {
        const exact = byName.get(c);
        if (exact) return exact.id;
      }
      for (const m of onThisDisk) {
        const nm = normName(m.name || '');
        if (nm && candidates.some(c => nm.startsWith(c) || c.startsWith(nm) || nm.includes(c))) {
          return m.id;
        }
      }
      return undefined;
    };

    const mk = (s: any): PartCard => ({
      kind: s.kind,
      name: s.name || s.label || (s.kind === 'P' ? 'Primaria' : 'Lógica'),
      start: s.start,
      end: s.end,
      size: s.size,
      percent: s.percent,
      mountId: findMountId(s),
    });

    return [...prim.map(mk), ...logi.map(mk)];
  }

  private buildPartsLoose(rep: DiskReport, mounts: MountView[]): PartCard[] {
    const prim = (rep?.segments || []).filter((s: any) => s?.kind === 'P');
    const logi = (rep?.extended?.segments || []).filter((s: any) => s?.kind === 'L');

    const normPath = (p: string) => (p || '').replace(/\\+/g, '/').toLowerCase().trim();
    const onThisDisk = (mounts || []).filter(
      (m) => normPath(m.diskPath) === normPath(rep?.diskPath || '')
    );

    const take = (list: any[], labelKind: 'P' | 'L') =>
      list.map((s: any, i: number): PartCard => ({
        kind: s.kind,
        name: s.name || s.label || (labelKind === 'P' ? `Primaria ${i + 1}` : `Lógica ${i + 1}`),
        start: s.start,
        end: s.end,
        size: s.size,
        percent: s.percent,
        mountId: onThisDisk[i]?.id,
      }));

    return [...take(prim, 'P'), ...take(logi, 'L')];
  }
}
