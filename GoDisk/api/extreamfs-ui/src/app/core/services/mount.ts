import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { map } from 'rxjs/operators';
import { Observable } from 'rxjs';

export interface MountView {
  id: string;
  diskPath: string;
  name?: string;
  start?: number;      
  size?: number;       
}

@Injectable({ providedIn: 'root' })
export class MountsService {
  private http = inject(HttpClient);
  private base = '/api';

  getAll(): Observable<MountView[]> {
    return this.http.get<any[]>(`${this.base}/mounts`, {
      params: { t: Date.now().toString() }
    }).pipe(
      map(list => Array.isArray(list) ? list : []),
      map(list => list.map(v => ({
        id: String(v.id ?? v.ID ?? ''),
        diskPath: String(v.diskPath ?? v.disk_path ?? v.DiskPath ?? ''),
        name: (v.name ?? v.partName ?? v.part_name ?? v.PartName ?? '').toString(),
        start: Number(v.start ?? v.Start ?? 0),
        size: Number(v.size ?? v.Size ?? 0),
      }) as MountView))
    );
  }
}
