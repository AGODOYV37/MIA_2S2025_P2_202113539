import { Component, OnDestroy, OnInit, NgZone, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { Reports, LSReport } from '../../core/services/reports';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-visualizador-part',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './part.html',
  styleUrls: ['./part.scss']
})
export class VisualizadorPartComponent implements OnInit, OnDestroy {
  id = '';
  loading = true;
  error = '';
  ls: LSReport | null = null;

  private sub?: Subscription;

  constructor(
    private route: ActivatedRoute,
    private reports: Reports,
    private zone: NgZone,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit(): void {
    this.sub = this.route.paramMap.subscribe(pm => {
      const id = pm.get('id') || '';
      if (!id) return;

      this.zone.run(() => {
        this.id = id;
        this.loading = true;
        this.error = '';
        this.ls = null;
        this.cdr.detectChanges();
      });

      // Carga LS del root
      this.reports.getLSWithRetry(id, '/', 2, 250).subscribe({
        next: (ls) => {
          this.zone.run(() => {
            this.ls = ls;
            this.loading = false;
            this.cdr.detectChanges();
          });
        },
        error: (err) => {
          this.zone.run(() => {
            this.error = (err?.error?.error || err?.message || 'Error cargando LS');
            this.loading = false;
            this.cdr.detectChanges();
          });
        }
      });
    });
  }

  ngOnDestroy(): void {
    this.sub?.unsubscribe();
  }
}
